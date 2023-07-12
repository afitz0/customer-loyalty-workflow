package loyalty

import (
	"context"
	"testing"
	"time"

	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"go.uber.org/zap/zapcore"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/afitz0/customer-loyalty-workflow/go/status"
	"github.com/afitz0/customer-loyalty-workflow/go/zapadapter"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUnitTestSuite(t *testing.T) {
	s := new(UnitTestSuite)
	logger := zapadapter.NewZapAdapter(zapadapter.NewZapLogger(zapcore.WarnLevel))
	s.SetLogger(logger)
	suite.Run(t, s)
}

func (s *UnitTestSuite) Test_Workflow() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	customer := CustomerInfo{
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		AccountActive: true,
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, 0)
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
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
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func (s *UnitTestSuite) Test_AddPointsForSinglePromo() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddPoints, status.Levels[1].MinimumPoints)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		s.NoError(err)

		var state GetStatusResponse
		err = result.Get(&state)
		s.NoError(err)
		s.Equal(status.Levels[1].MinimumPoints, state.Points)
		s.Equal(status.Levels[1], state.Tier)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func (s *UnitTestSuite) Test_AddPointsForMultiPromo() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	targetLevel := len(status.Levels) - 1
	targetTier := status.Levels[targetLevel]

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
		s.Equal(targetTier, state.Tier)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
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
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
	env.AssertCalled(s.T(), "SendEmail", mock.Anything, EmailCancelAccount)
}

func (s *UnitTestSuite) Test_InviteGuest() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	// first, invite the guest. This should result in another workflow being started
	env.RegisterDelayedCallback(func() {
		guestID := "guest"
		env.SignalWorkflow(SignalInviteGuest, guestID)
	}, 0)

	// then, see if we can query it
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflowByID(
			CustomerWorkflowID("guest"),
			QueryGetStatus)
		s.NoError(err)
		var state GetStatusResponse
		err = result.Get(&state)
		s.NoError(err)
		s.True(state.AccountActive)
	}, time.Second*1)

	// cancel workflows to not timeout
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
		err := env.SignalWorkflowByID(
			CustomerWorkflowID("guest"),
			SignalCancelAccount, nil)
		s.NoError(err)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   1,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func (s *UnitTestSuite) Test_QueryGuests() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

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
		err := env.SignalWorkflowByID(
			CustomerWorkflowID("guest"),
			SignalCancelAccount, nil)
		s.NoError(err)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   1,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func (s *UnitTestSuite) Test_InviteGuestPreviouslyCanceled() {
	env := s.NewTestWorkflowEnvironment()

	env.RegisterActivity(&Activities{})
	env.OnActivity("SendEmail", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, body string) (err error) {
			return nil
		})

	order := time.Second
	guestID := "guest"
	guestWfID := CustomerWorkflowID(guestID)

	// first, invite the guest. This should result in another workflow being started
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalInviteGuest, guestID)
	}, order)
	order += time.Second

	// immediately cancel the guest account. this should keep it queryable, but sets AccountActive -> false
	env.RegisterDelayedCallback(func() {
		err := env.SignalWorkflowByID(
			guestWfID,
			SignalCancelAccount, nil)
		s.NoError(err)
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
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   2,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	s.NoError(env.GetWorkflowResult(nil))

	env.AssertCalled(s.T(), "SendEmail", mock.Anything, EmailGuestCanceled)
}

func (s *UnitTestSuite) Test_SendEmailActivity() {
	env := s.NewTestActivityEnvironment()

	var a *Activities
	env.RegisterActivity(a)

	_, err := env.ExecuteActivity(a.SendEmail, "Hello, World!")
	s.NoError(err)
}

func (s *UnitTestSuite) Test_SimpleReplay() {
	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(CustomerLoyaltyWorkflow)
	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "simple_replay.json")
	s.NoError(err)
}
