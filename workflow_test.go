package starter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/afitz0/customer-loyalty-workflow/common"
	"github.com/afitz0/customer-loyalty-workflow/status"
)

func Test_Workflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		AccountActive: true,
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalCancelAccount, nil)
	}, 0)
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func Test_AddPoints(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalAddPoints, 100)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(common.QueryGetStatus)
		require.NoError(t, err)

		var state GetStatusResponse
		err = result.Get(&state)
		require.NoError(t, err)
		require.Equal(t, 100, state.Points)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func Test_AddPointsForSinglePromo(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalAddPoints, status.Levels[1].MinimumPoints)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(common.QueryGetStatus)
		require.NoError(t, err)

		var state GetStatusResponse
		err = result.Get(&state)
		require.NoError(t, err)
		require.Equal(t, status.Levels[1].MinimumPoints, state.Points)
		require.Equal(t, status.Levels[1], state.Tier)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func Test_AddPointsForMultiPromo(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	targetLevel := len(status.Levels) - 1
	targetTier := status.Levels[targetLevel]

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalAddPoints, targetTier.MinimumPoints)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(common.QueryGetStatus)
		require.NoError(t, err)

		var state GetStatusResponse
		err = result.Get(&state)
		require.NoError(t, err)
		require.Equal(t, targetTier.MinimumPoints, state.Points)
		require.Equal(t, targetTier, state.Tier)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func Test_CancelAccount(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterActivity(&Activities{})
	env.OnActivity("SendEmail", mock.Anything, mock.Anything).Return(nil)

	// cancel account
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalCancelAccount, nil)
	}, time.Second*1)

	// check status
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(common.QueryGetStatus)
		require.NoError(t, err)
		var statusResponse GetStatusResponse
		err = result.Get(&statusResponse)
		require.NoError(t, err)

		require.False(t, statusResponse.AccountActive)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
	env.AssertCalled(t, "SendEmail", mock.Anything, common.EmailCancelAccount)
}

func Test_InviteGuest(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	// first, invite the guest. This should result in another workflow being started
	env.RegisterDelayedCallback(func() {
		guestId := "guest"
		env.SignalWorkflow(common.SignalInviteGuest, guestId)
	}, 0)

	// then, see if we can query it
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflowByID(
			fmt.Sprintf(common.CustomerWorkflowIdFormat, "guest"),
			common.QueryGetStatus)
		require.NoError(t, err)
		var state GetStatusResponse
		err = result.Get(&state)
		require.NoError(t, err)
		require.True(t, state.AccountActive)
	}, time.Second*1)

	// cancel workflows to not timeout
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalCancelAccount, nil)
		err := env.SignalWorkflowByID(
			fmt.Sprintf(common.CustomerWorkflowIdFormat, "guest"),
			common.SignalCancelAccount, nil)
		require.NoError(t, err)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   1,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func Test_QueryGuests(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	guestId := "guest"

	// first, invite the guest. This should result in this guest's ID being added to the original customer
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalInviteGuest, guestId)
	}, 0)

	// Query the original customer's guest list
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(common.QueryGetGuests)
		require.NoError(t, err)
		var guests []string
		err = result.Get(&guests)
		require.NoError(t, err)
		require.Equal(t, 1, len(guests))
		require.Equal(t, guestId, guests[0])
	}, time.Second*1)

	// cancel workflows to not timeout
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalCancelAccount, nil)
		err := env.SignalWorkflowByID(
			fmt.Sprintf(common.CustomerWorkflowIdFormat, "guest"),
			common.SignalCancelAccount, nil)
		require.NoError(t, err)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   1,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func Test_InviteGuestPreviouslyCanceled(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterActivity(&Activities{})
	env.OnActivity("SendEmail", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, body string) (err error) {
			return nil
		})

	order := time.Second
	guestId := "guest"
	guestWfId := fmt.Sprintf(common.CustomerWorkflowIdFormat, guestId)

	// first, invite the guest. This should result in another workflow being started
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalInviteGuest, guestId)
	}, order)
	order += time.Second

	// immediately cancel the guest account. this should keep it queryable, but sets AccountActive -> false
	env.RegisterDelayedCallback(func() {
		err := env.SignalWorkflowByID(
			guestWfId,
			common.SignalCancelAccount, nil)
		require.NoError(t, err)
	}, order)
	order += time.Second

	// then, try to invite them again. the "guest has already canceled" email should be sent
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalInviteGuest, guestId)
	}, order)
	order += time.Second

	// cancel original workflow to not timeout
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(common.SignalCancelAccount, nil)
	}, order)
	order += time.Second

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   2,
		Name:          "Customer",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.NoError(t, env.GetWorkflowResult(nil))

	env.AssertCalled(t, "SendEmail", mock.Anything, common.EmailGuestCanceled)
}

func Test_SendEmailActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	var a *Activities
	env.RegisterActivity(a)

	_, err := env.ExecuteActivity(a.SendEmail, "Hello, World!")
	require.NoError(t, err)
}

func Test_SimpleReplay(t *testing.T) {
	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(CustomerLoyaltyWorkflow)
	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "simple_replay.json")
	require.NoError(t, err)
}
