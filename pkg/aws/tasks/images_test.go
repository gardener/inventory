// TODO switch to blackbox (tasks_testing)
package tasks

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"
)

// TODO: This will have to be enabled after payloads for the collect-all task are implemented in the future.

func TestNewCollectImagesForRegionTaskCorrectPath(t *testing.T) {
	payload := CollectImagesForRegionPayload{
		//Test must catch this.
		Region:      "valid-region",
		ImageOwners: []int64{1, 2, 3},
	}

	task, err := NewCollectImagesForRegionTask(payload)
	if err != nil {
		t.Errorf("Task creation should not error out.")
	}

	if task == nil {
		t.Errorf("Task is nil. Should be valid task.")
	}
}

func TestNewCollectImagesForRegionTaskValidatesInput(t *testing.T) {
	// empty Region case
	payload := CollectImagesForRegionPayload{
		//Test must catch this.
		Region: "",
	}

	_, err := NewCollectImagesForRegionTask(payload)
	if err == nil {
		t.Errorf("Region should be validated on new task creation.")
	}

	// nil ImageOwners case
	var nilOwners []int64
	payload = CollectImagesForRegionPayload{
		//Test must catch this.
		Region:      "example-region",
		ImageOwners: nilOwners,
	}

	_, err = NewCollectImagesForRegionTask(payload)
	if err == nil {
		t.Errorf("ImageOwners should be checked for nil on new task creation.")
	}

	// empty ImageOwners case
	var emptyOwners []int64
	payload = CollectImagesForRegionPayload{
		//Test must catch this.
		Region:      "example-region",
		ImageOwners: emptyOwners,
	}

	_, err = NewCollectImagesForRegionTask(payload)

	if err == nil {
		t.Errorf("ImageOwners should be checked for empty on new task creation.")
	}
}

func TestHandleCollectImagesForRegionTaskValidatesNilPayload(t *testing.T) {
	testTask := asynq.NewTask(AWS_COLLECT_IMAGES_REGION_TYPE, nil)

	if testTask == nil {
		t.Errorf("Task creation failed.")
	}

	if err := HandleCollectImagesForRegionTask(context.Background(), testTask); err == nil {
		t.Errorf("Unmarshalling nil payload should fail.")
	}
}

func TestHandleCollectImagesForRegionTaskValidatesPayload(t *testing.T) {
	nilOwnerPayload := CollectImagesForRegionPayload{
		Region: "test-region",
	}

	rawPayload, err := yaml.Marshal(nilOwnerPayload)
	if err != nil {
		t.Fatal(err)
	}
	testTask := asynq.NewTask(AWS_COLLECT_IMAGES_REGION_TYPE, rawPayload)

	err = HandleCollectImagesForRegionTask(context.Background(), testTask)
	if err == nil {
		t.Errorf("ImageOwners should be checked for nil on handling task.")
	}

	emptyOwner := CollectImagesForRegionPayload{
		Region:      "test-region",
		ImageOwners: []int64{},
	}

	rawPayload, err = yaml.Marshal(emptyOwner)
	if err != nil {
		t.Fatal(err)
	}

	testTask = asynq.NewTask(AWS_COLLECT_IMAGES_REGION_TYPE, rawPayload)

	err = HandleCollectImagesForRegionTask(context.Background(), testTask)
	if err == nil {
		t.Errorf("ImageOwners should be checked for empty value on handling task.")
	}

	emptyRegion := CollectImagesForRegionPayload{
		Region:      "",
		ImageOwners: []int64{1},
	}

	rawPayload, err = yaml.Marshal(emptyRegion)
	if err != nil {
		t.Fatal(err)
	}
	testTask = asynq.NewTask(AWS_COLLECT_IMAGES_REGION_TYPE, rawPayload)

	err = HandleCollectImagesForRegionTask(context.Background(), testTask)

	if err == nil {
		t.Errorf("Region should be checked for empty value on new task creation.")
	}
}

func TestHandleCollectImagesTaskValidatesPayload(t *testing.T) {
	nilOwnerPayload := CollectImagesPayload{}

	rawPayload, err := yaml.Marshal(nilOwnerPayload)
	if err != nil {
		t.Fatal(err)
	}
	testTask := asynq.NewTask(AWS_COLLECT_IMAGES_TYPE, rawPayload)

	err = HandleCollectImagesTask(context.Background(), testTask)
	if err == nil {
		t.Errorf("ImageOwners should be checked for nil on handling task.")
	}

	emptyOwner := CollectImagesPayload{
		ImageOwners: []int64{},
	}

	rawPayload, err = yaml.Marshal(emptyOwner)
	if err != nil {
		t.Fatal(err)
	}
	testTask = asynq.NewTask(AWS_COLLECT_IMAGES_TYPE, rawPayload)

	err = HandleCollectImagesTask(context.Background(), testTask)

	if err == nil {
		t.Errorf("ImageOwners should be checked for empty value on handling task.")
	}
}
