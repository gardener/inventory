// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/azure/models"
	azureutils "github.com/gardener/inventory/pkg/azure/utils"
	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/clients/db"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// TaskCollectUsers is the name of the task for collecting Microsoft Entra user
// accounts.
const TaskCollectUsers = "az:task:collect-users"

// CollectUsersPayload is the payload used for collecting Microsoft Entra user
// accounts.
type CollectUsersPayload struct {
	// TenantID specifies the Azure Tenant ID from which to
	// collect.
	TenantID string `json:"tenant_id" yaml:"tenant_id"`

	// UserPrincipalName specifies the principal name of the user to be
	// collected.
	UserPrincipalName string `json:"user_principal_name" yaml:"user_principal_name"`
}

// HandleCollectUsersTask is the handler, which collects Microsoft Entra user
// accounts.
func HandleCollectUsersTask(ctx context.Context, t *asynq.Task) error {
	data := t.Payload()
	if data == nil {
		return asynqutils.SkipRetry(ErrNoPayload)
	}

	var payload CollectUsersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.TenantID == "" {
		return asynqutils.SkipRetry(ErrNoTenantID)
	}
	if payload.UserPrincipalName == "" {
		return asynqutils.SkipRetry(ErrNoUserPrincipalName)
	}

	return collectUsers(ctx, payload)
}

// collectUsers collects the Microsoft Entra users specified in the given
// payload.
func collectUsers(ctx context.Context, payload CollectUsersPayload) error {
	client, ok := azureclients.GraphClientset.Get(payload.TenantID)
	if !ok {
		return asynqutils.SkipRetry(fmt.Errorf("client not found for tenant id %s", payload.TenantID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting Azure user",
		"tenant_id", payload.TenantID,
		"user_principal_name", payload.UserPrincipalName,
	)

	result, err := client.Client.UsersWithUserPrincipalName(ptr.To(payload.UserPrincipalName)).Get(ctx, nil)
	if err != nil {
		logger.Error(
			"failed to get Azure user",
			"tenant_id", payload.TenantID,
			"user_principal_name", payload.UserPrincipalName,
			"reason", err,
		)

		return azureutils.MaybeSkipRetry(err)
	}

	userID := ptr.Value(result.GetId(), "")
	if userID == "" {
		logger.Warn(
			"empty id received for user",
			"tenant_id", payload.TenantID,
			"user_principal_name", payload.UserPrincipalName,
		)

		return nil
	}

	mail := ptr.Value(result.GetMail(), "")
	if mail == "" {
		logger.Warn(
			"empty mail received for user",
			"tenant_id", payload.TenantID,
			"user_principal_name", payload.UserPrincipalName,
		)

		return nil
	}

	user := models.User{
		TenantID: payload.TenantID,
		UserID:   userID,
		Mail:     mail,
	}

	out, err := db.DB.NewInsert().
		Model(&user).
		On("CONFLICT (tenant_id, user_id) DO UPDATE").
		Set("mail = EXCLUDED.mail").
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
		"populated azure user",
		"count", count,
		"tenant_id", payload.TenantID,
		"user_principal_name", payload.UserPrincipalName,
	)

	return nil
}
