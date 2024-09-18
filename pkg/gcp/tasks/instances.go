// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"github.com/hibiken/asynq"
	"google.golang.org/api/iterator"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/constants"
	"github.com/gardener/inventory/pkg/gcp/models"
	"github.com/gardener/inventory/pkg/gcp/utils"
	gcputils "github.com/gardener/inventory/pkg/gcp/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// TaskCollectInstances is the name of the task for collecting GCP Instances
const TaskCollectInstances = "gcp:task:collect-instances"

// CollectInstancesPayload is the payload used for collecting GCP Instances from
// a given GCP Project.
type CollectInstancesPayload struct {
	// ProjectID specifies the globally unique project id from which to
	// collect GCP Compute Engine Instances.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// ErrNoSourceMachineImage is an error, which is returned when a
// source machine image was not found for an instance.
var ErrNoSourceImage = errors.New("no source machine image found")

// ErrNoDiskInformation is an error, which is returned when
// disk information was not found for an instance.
var ErrNoDiskInformation = errors.New("no disk information found")

// NewCollectInstancesTask creates a new [asynq.Task] for collecting GCP Compute
// Engine Instances, without specifying a payload.
func NewCollectInstancesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectInstances, nil)
}

// HandleCollectInstancesTask is the handler, which collects GCP Instances.
func HandleCollectInstancesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting Compute Engine Instances from all registered projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectInstances(ctx)
	}

	var payload CollectInstancesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectInstances(ctx, payload)
}

// enqueueCollectInstances enqueues tasks for collecting GCP Compute Engine
// Instances for all known projects.
func enqueueCollectInstances(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if gcpclients.InstancesClientset.Length() == 0 {
		logger.Warn("no GCP instance clients found")
		return nil
	}

	// Enqueue tasks for all registered GCP Projects
	err := gcpclients.InstancesClientset.Range(func(projectID string, _ *gcpclients.Client[*compute.InstancesClient]) error {
		payload := CollectInstancesPayload{
			ProjectID: projectID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP Compute Instances",
				"project", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}
		task := asynq.NewTask(TaskCollectInstances, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"project", projectID,
		)

		return nil
	})

	return err
}

// collectInstances collects the GCP Compute Engine instances from the project
// specified in the payload.
func collectInstances(ctx context.Context, payload CollectInstancesPayload) error {
	client, ok := gcpclients.InstancesClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting GCP instances", "project", payload.ProjectID)

	pageSize := uint32(constants.PageSize)
	partialSuccess := true
	req := &computepb.AggregatedListInstancesRequest{
		Project:              gcputils.ProjectFQN(payload.ProjectID),
		MaxResults:           &pageSize,
		ReturnPartialSuccess: &partialSuccess,
	}

	instances := make([]models.Instance, 0)
	nics := make([]models.NetworkInterface, 0)
	it := client.Client.AggregatedList(ctx, req)
	for {
		// The iterator returns a k/v pair, where the key represents a
		// specific GCP zone and the value is the slice of instances in
		// that zone.
		pair, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			logger.Error(
				"failed to get GCP instances",
				"project", payload.ProjectID,
				"reason", err,
			)
			return err
		}

		zone := gcputils.UnqualifyZone(pair.Key)
		region := gcputils.RegionFromZone(zone)
		for _, inst := range pair.Value.Instances {
			sourceMachineImage, err := getSourceMachineImageFromDisks(ctx, payload.ProjectID, zone, inst.GetDisks())
			if err != nil {
				logger.Error(
					"could not get source machine image",
					"reason",
					err,
				)
			}

			// Collect instance
			instance := models.Instance{
				Name:                 inst.GetName(),
				Hostname:             inst.GetHostname(),
				InstanceID:           inst.GetId(),
				ProjectID:            payload.ProjectID,
				Zone:                 zone,
				Region:               region,
				CanIPForward:         inst.GetCanIpForward(),
				CPUPlatform:          inst.GetCpuPlatform(),
				CreationTimestamp:    inst.GetCreationTimestamp(),
				Description:          inst.GetDescription(),
				LastStartTimestamp:   inst.GetLastStartTimestamp(),
				LastStopTimestamp:    inst.GetLastStopTimestamp(),
				LastSuspendTimestamp: inst.GetLastSuspendedTimestamp(),
				MachineType:          utils.ResourceNameFromURL(inst.GetMachineType()),
				MinCPUPlatform:       inst.GetMinCpuPlatform(),
				SelfLink:             inst.GetSelfLink(),
				SourceMachineImage:   sourceMachineImage,
				Status:               inst.GetStatus(),
				StatusMessage:        inst.GetStatusMessage(),
			}
			instances = append(instances, instance)

			// Collect NICs
			for _, ni := range inst.GetNetworkInterfaces() {
				nic := models.NetworkInterface{
					Name:           ni.GetName(),
					ProjectID:      payload.ProjectID,
					InstanceID:     inst.GetId(),
					Network:        gcputils.ResourceNameFromURL(ni.GetNetwork()),
					Subnetwork:     gcputils.ResourceNameFromURL(ni.GetSubnetwork()),
					IPv4:           net.ParseIP(ni.GetNetworkIP()),
					IPv6:           net.ParseIP(ni.GetIpv6Address()),
					IPv6AccessType: ni.GetIpv6AccessType(),
					NICType:        ni.GetNicType(),
					StackType:      ni.GetStackType(),
				}
				nics = append(nics, nic)
			}
		}
	}

	// Upsert instances
	if len(instances) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&instances).
		On("CONFLICT (project_id, instance_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("hostname = EXCLUDED.hostname").
		Set("zone = EXCLUDED.zone").
		Set("region = EXCLUDED.region").
		Set("can_ip_forward = EXCLUDED.can_ip_forward").
		Set("cpu_platform = EXCLUDED.cpu_platform").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("description = EXCLUDED.description").
		Set("last_start_timestamp = EXCLUDED.last_start_timestamp").
		Set("last_stop_timestamp = EXCLUDED.last_stop_timestamp").
		Set("last_suspend_timestamp = EXCLUDED.last_suspend_timestamp").
		Set("machine_type = EXCLUDED.machine_type").
		Set("min_cpu_platform = EXCLUDED.min_cpu_platform").
		Set("self_link = EXCLUDED.self_link").
		Set("source_machine_image = EXCLUDED.source_machine_image").
		Set("status = EXCLUDED.status").
		Set("status_message = EXCLUDED.status_message").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gcp instances",
		"project", payload.ProjectID,
		"count", count,
	)

	// Upsert NICs
	if len(nics) == 0 {
		return nil
	}

	out, err = db.DB.NewInsert().
		Model(&nics).
		On("CONFLICT (project_id, instance_id, name) DO UPDATE").
		Set("network = EXCLUDED.network").
		Set("subnetwork = EXCLUDED.subnetwork").
		Set("ipv4 = EXCLUDED.ipv4").
		Set("ipv6 = EXCLUDED.ipv6").
		Set("ipv6_access_type = EXCLUDED.ipv6_access_type").
		Set("nic_type = EXCLUDED.nic_type").
		Set("stack_type = EXCLUDED.stack_type").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gcp network interfaces",
		"project", payload.ProjectID,
		"count", count,
	)

	return nil
}

func getSourceMachineImageFromDisks(
	ctx context.Context,
	projectID string,
	zone string,
	disks []*computepb.AttachedDisk,
) (string, error) {
	// Iterate through disks, find first boot disk.
	// Derive source machine image from it.

	// expectations:
	// only 1 boot disk per instance (uses first one)
	// the attached disk points to another disk, which contains the source machine image

	var sourceMachineImage string

	for _, disk := range disks {
		if disk == nil {
			continue
		}

		if !disk.GetBoot() {
			continue
		}

		sourceDisk := disk.GetSource()
		sourceDiskName := utils.ResourceNameFromURL(sourceDisk)

		diskClient, ok := gcpclients.DisksClientset.Get(projectID)
		if !ok {
			return "", asynqutils.SkipRetry(ClientNotFound(projectID))
		}

		diskRequest := computepb.GetDiskRequest{
			Project: projectID,
			Zone:    zone,
			Disk:    sourceDiskName,
		}
		disk, err := diskClient.Client.Get(ctx, &diskRequest)
		if err != nil {
			return "", ErrNoDiskInformation
		}

		sourceMachineImage = utils.ResourceNameFromURL(disk.GetSourceImage())

		return sourceMachineImage, nil
	}

	return sourceMachineImage, nil
}
