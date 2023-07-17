package loyalty

import (
	"context"
	"fmt"
	"go.temporal.io/sdk/temporal"
	"testing"
	"time"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"go.uber.org/zap/zapcore"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUnitTestSuite(t *testing.T) {
	s := new(UnitTestSuite)
	logger := NewZapAdapter(NewZapLogger(zapcore.WarnLevel))
	s.SetLogger(logger)
	suite.Run(t, s)
}

func (s *UnitTestSuite) Test_Workflow() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	customer := CustomerInfo{
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   StatusLevels[0],
		Name:          "Customer",
		AccountActive: true,
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, 0)
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer, true)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))
}

func (s *UnitTestSuite) Test_AddPoints() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddPoints, 100)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		s.NoError(err)

		var state GetStatusResponse
		err = result.Get(&state)
		s.NoError(err)
		s.Equal(100, state.Points)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		StatusLevel:   StatusLevels[0],
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer, true)
	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))
}

func (s *UnitTestSuite) Test_AddPointsForSinglePromo() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddPoints, StatusLevels[1].MinimumPoints)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		s.NoError(err)

		var state GetStatusResponse
		err = result.Get(&state)
		s.NoError(err)
		s.Equal(StatusLevels[1].MinimumPoints, state.Points)
		s.Equal(StatusLevels[1], &state.StatusLevel)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		StatusLevel:   StatusLevels[0],
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer, true)
	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))
}

func (s *UnitTestSuite) Test_AddPointsForMultiPromo() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	targetLevel := len(StatusLevels) - 1
	targetTier := StatusLevels[targetLevel]

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddPoints, targetTier.MinimumPoints)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		s.NoError(err)

		var state GetStatusResponse
		err = result.Get(&state)
		s.NoError(err)
		s.Equal(targetTier.MinimumPoints, state.Points)
		s.Equal(targetTier, &state.StatusLevel)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		StatusLevel:   StatusLevels[0],
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer, true)
	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))
}

func (s *UnitTestSuite) Test_CancelAccount() {
	env := s.NewTestWorkflowEnvironment()

	env.RegisterActivity(&Activities{})
	env.OnActivity("SendEmail", mock.Anything, mock.Anything).Return(nil)

	// cancel account
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*1)

	// check status
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		s.NoError(err)
		var statusResponse GetStatusResponse
		err = result.Get(&statusResponse)
		s.NoError(err)
		s.False(statusResponse.AccountActive)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerID:    "123",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer, true)
	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))

	env.AssertCalled(s.T(), "SendEmail", mock.Anything, emailCancelAccount)
}

func (s *UnitTestSuite) Test_InviteGuest() {
	env := s.NewTestWorkflowEnvironment()
	childEnv := s.NewTestWorkflowEnvironment()

	a := &Activities{}
	env.RegisterActivity(a)
	childEnv.RegisterActivity(a)

	env.OnActivity(a.SendEmail, mock.Anything, mock.Anything).
		Return(nil)
	childEnv.OnActivity(a.SendEmail, mock.Anything, mock.Anything).
		Return(nil)

	env.OnActivity(a.StartGuestWorkflow, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			guest := args.Get(1).(CustomerInfo)
			childEnv.ExecuteWorkflow(CustomerLoyaltyWorkflow, guest, true)
		}).
		Return(nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalInviteGuest, "guest")
	}, 0)

	env.RegisterDelayedCallback(func() {
		val, err := childEnv.QueryWorkflow(QueryGetStatus)
		s.NoError(err)
		var r GetStatusResponse
		err = val.Get(&r)
		s.NoError(err)
		s.True(r.AccountActive)
	}, time.Second)

	// cancel workflows to not timeout
	env.RegisterDelayedCallback(func() {
		childEnv.SignalWorkflow(SignalCancelAccount, nil)
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerID:    "host",
		StatusLevel:   StatusLevels[2],
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer, true)
	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))

	env.AssertCalled(s.T(), "SendEmail", mock.Anything, emailGuestInvited)
	env.AssertNotCalled(s.T(), "SendEmail", mock.Anything, emailInsufficientPoints)

	childEnv.AssertCalled(s.T(), "SendEmail", mock.Anything, fmt.Sprintf(emailWelcome, StatusLevels[1].Name))
}

func (s *UnitTestSuite) Test_QueryGuests() {
	env := s.NewTestWorkflowEnvironment()

	a := &Activities{}
	env.RegisterActivity(a)

	env.OnActivity(a.StartGuestWorkflow, mock.Anything, mock.Anything).Return(nil)

	guestID := "guest"

	// first, invite the guest. This should result in this guest's ID being added to the original customer
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalInviteGuest, guestID)
	}, 0)

	// Query the original customer's guest list
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetGuests)
		s.NoError(err)
		var guests []string
		err = result.Get(&guests)
		s.NoError(err)
		s.Equal(1, len(guests))
		s.Equal(guestID, guests[0])
	}, time.Second*1)

	// cancel workflows to not timeout
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		StatusLevel:   StatusLevels[1],
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer, true)
	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))
}

func (s *UnitTestSuite) Test_InviteGuestPreviouslyCanceled() {
	env := s.NewTestWorkflowEnvironment()

	a := &Activities{}
	env.RegisterActivity(a)

	env.OnActivity(a.SendEmail, mock.Anything, mock.Anything).Return(nil)

	call := 0
	env.OnActivity(a.StartGuestWorkflow, mock.Anything, mock.Anything).
		Twice().
		Return(func(_ context.Context, _ CustomerInfo) error {
			if call == 0 {
				call++
				return nil
			} else {
				return temporal.NewApplicationError("", "GuestAlreadyCanceledError")
			}
		})

	order := time.Second
	guestID := "guest"

	// first, invite the guest. This should result in another workflow being started
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalInviteGuest, guestID)
	}, order)
	order += time.Second

	// then, try to invite them again. the "guest has already canceled" email should be sent
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalInviteGuest, guestID)
	}, order)
	order += time.Second

	// cancel original workflow to not timeout
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, order)
	order += time.Second

	customer := CustomerInfo{
		StatusLevel:   StatusLevels[2],
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer, true)
	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))

	env.AssertCalled(s.T(), "SendEmail", mock.Anything, emailGuestInvited)
	env.AssertCalled(s.T(), "SendEmail", mock.Anything, emailGuestCanceled)
}

func (s *UnitTestSuite) Test_SendEmailActivity() {
	env := s.NewTestActivityEnvironment()

	var a *Activities
	env.RegisterActivity(a)

	_, err := env.ExecuteActivity(a.SendEmail, "Hello, World!")
	s.NoError(err)
}

func (s *UnitTestSuite) Test_SimpleReplay() {
	replayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{
		DataConverter: converter.GetDefaultDataConverter(),
	})
	s.NoError(err)

	replayer.RegisterWorkflow(CustomerLoyaltyWorkflow)
	err = replayer.ReplayWorkflowHistoryFromJSONFile(nil, "simple_replay.json")
	s.NoError(err)
}
