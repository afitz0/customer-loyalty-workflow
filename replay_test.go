package starter

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/api/workflowservicemock/v1"
	"go.temporal.io/sdk/worker"
)

type replayTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
	service  *workflowservicemock.MockWorkflowServiceClient
}

func TestReplayTestSuite(t *testing.T) {
	s := new(replayTestSuite)
	suite.Run(t, s)
}

func (s *replayTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.service = workflowservicemock.NewMockWorkflowServiceClient(s.mockCtrl)
}

func (s *replayTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

// Download the replay json with:
// [Version 1:]
//      tctl wf show -w starter-workflow --output_filename ./starter_replay.json
// [Version 2:] TODO: update when tctl v2 is out of beta
// [Temporal UI:]
//      click download
func (s *replayTestSuite) TestReplayWorkflowHistoryFromFile() {
	replayer := worker.NewWorkflowReplayer()

	replayer.RegisterWorkflow(Workflow)

	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "starter_replay.json")
	require.NoError(s.T(), err)
}
