package tasks

import (
	"context"
	"encoding/json"
	"net"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/openstack/models"
	openstackutils "github.com/gardener/inventory/pkg/openstack/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectRouters is the name of the task for collecting
	// OpenStack Routers.
	TaskCollectRouters = "openstack:task:collect-routers"
)

// CollectRoutersPayload represents the payload, which specifies
// where to collect OpenStack Routers from.
type CollectRoutersPayload struct {
	// Scope specifies the client scope for which to collect.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectRoutersTask creates a new [asynq.Task] for collecting OpenStack
// Routers, without specifying a payload.
func NewCollectRoutersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectRouters, nil)
}

// HandleCollectRoutersTask handles the task for collecting OpenStack Routers.
func HandleCollectRoutersTask(ctx context.Context, t *asynq.Task) error {
	data := t.Payload()
	if data == nil {
		return enqueueCollectRouters(ctx)
	}

	var payload CollectRoutersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectRouters(ctx, payload)
}

// enqueueCollectRouters enqueues tasks for collecting OpenStack Routers from
// all configured OpenStack projects by creating a payload with the respective
// client scope
func enqueueCollectRouters(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.NetworkClientset.Length() == 0 {
		logger.Warn("no OpenStack network clients found")
		return nil
	}

	return openstackclients.NetworkClientset.Range(func(scope openstackclients.ClientScope, client openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectRoutersPayload{
			Scope: scope,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack routers",
				"project", scope.Project,
				"domain", scope.Domain,
				"region", scope.Region,
				"reason", err,
			)
			return err
		}

		task := asynq.NewTask(TaskCollectRouters, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project", scope.Project,
				"domain", scope.Domain,
				"region", scope.Region,
				"reason", err,
			)
			return err
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"project", scope.Project,
			"domain", scope.Domain,
			"region", scope.Region,
		)

		return nil
	})
}

// collectRouters collects the OpenStack Routers from the specified project,
// using the client associated with the project in the given payload.
func collectRouters(ctx context.Context, payload CollectRoutersPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.NetworkClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack routers",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"named_credentials", payload.Scope.NamedCredentials,
	)

	items := make([]models.Router, 0)
	externalIPs := make([]models.RouterExternalIP, 0)

	err := routers.List(client.Client, routers.ListOpts{}).
		EachPage(ctx,
			func(ctx context.Context, page pagination.Page) (bool, error) {
				routerList, err := routers.ExtractRouters(page)
				if err != nil {
					logger.Error(
						"could not extract router pages",
						"reason", err,
					)
					return false, err
				}

				for _, router := range routerList {
					item := models.Router{
						RouterID:          router.ID,
						Name:              router.Name,
						ProjectID:         router.ProjectID,
						Domain:            payload.Scope.Domain,
						Region:            payload.Scope.Region,
						Status:            router.Status,
						Description:       router.Description,
						ExternalNetworkID: router.GatewayInfo.NetworkID,
					}

					items = append(items, item)
					for _, fixedIP := range router.GatewayInfo.ExternalFixedIPs {
						externalIP := net.ParseIP(fixedIP.IPAddress)

						if externalIP == nil {
							logger.Warn(
								"empty external IP record for router",
								"router id",
								router.ID,
							)
						}

						item := models.RouterExternalIP{
							RouterID:         router.ID,
							ExternalIP:       externalIP,
							ExternalSubnetID: fixedIP.SubnetID,
							ProjectID:        router.ProjectID,
						}

						externalIPs = append(externalIPs, item)
					}
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract router pages",
			"reason", err,
		)
		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (router_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("domain = EXCLUDED.domain").
		Set("region = EXCLUDED.region").
		Set("status = EXCLUDED.status").
		Set("description = EXCLUDED.description").
		Set("external_network_id = EXCLUDED.external_network_id").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert routers into db",
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack routers",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	if len(externalIPs) == 0 {
		return nil
	}

	out, err = db.DB.NewInsert().
		Model(&externalIPs).
		On("CONFLICT (router_id, external_ip, external_subnet_id, project_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert router external IPs into db",
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)
		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack router external IPs",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	return nil
}
