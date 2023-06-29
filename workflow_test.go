package starter

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"testing"
	"time"
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
		Guests:        []string{},
		AccountActive: true,
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
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
		env.SignalWorkflow(SignalAddPoints, 100)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		require.NoError(t, err)

		var state GetStatusResponse
		err = result.Get(&state)
		require.NoError(t, err)
		require.Equal(t, 100, state.Points)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		Guests:        []string{},
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func Test_AddPointsForSinglePromo(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddPoints, StatusTiers[1].MinimumPoints)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		require.NoError(t, err)

		var state GetStatusResponse
		err = result.Get(&state)
		require.NoError(t, err)
		require.Equal(t, StatusTiers[1].MinimumPoints, state.Points)
		require.Equal(t, 1, state.StatusLevel)
		require.Equal(t, StatusTiers[1], state.Tier)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		Guests:        []string{},
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
}

func Test_AddPointsForMultiPromo(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	targetLevel := len(StatusTiers) - 1
	targetTier := StatusTiers[targetLevel]

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalAddPoints, targetTier.MinimumPoints)
	}, 0)
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		require.NoError(t, err)

		var state GetStatusResponse
		err = result.Get(&state)
		require.NoError(t, err)
		require.Equal(t, targetTier.MinimumPoints, state.Points)
		require.Equal(t, targetLevel, state.StatusLevel)
		require.Equal(t, targetTier, state.Tier)
	}, time.Second*1)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		Guests:        []string{},
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
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, time.Second*1)

	// check status
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetStatus)
		require.NoError(t, err)
		var status GetStatusResponse
		err = result.Get(&status)
		require.NoError(t, err)

		require.False(t, status.AccountActive)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
	env.AssertCalled(t, "SendEmail", mock.Anything, EmailCancelAccount)
}

func Test_InviteGuest(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	// first, invite the guest. This should result in another workflow being started
	env.RegisterDelayedCallback(func() {
		guestId := "guest"
		env.SignalWorkflow(SignalInviteGuest, guestId)
	}, 0)

	// then, see if we can query it
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflowByID(
			fmt.Sprintf(CustomerWorkflowIdFormat, "guest"),
			QueryGetStatus)
		require.NoError(t, err)
		var state GetStatusResponse
		err = result.Get(&state)
		require.NoError(t, err)
		require.True(t, state.AccountActive)
	}, time.Second*1)

	// cancel workflows to not timeout
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
		env.SignalWorkflowByID(
			fmt.Sprintf(CustomerWorkflowIdFormat, "guest"),
			SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   1,
		Name:          "Customer",
		Guests:        []string{},
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
		env.SignalWorkflow(SignalInviteGuest, guestId)
	}, 0)

	// Query the original customer's guest list
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(QueryGetGuests)
		require.NoError(t, err)
		var guests []string
		err = result.Get(&guests)
		require.NoError(t, err)
		require.Equal(t, 1, len(guests))
		require.Equal(t, guestId, guests[0])
	}, time.Second*1)

	// cancel workflows to not timeout
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelAccount, nil)
		env.SignalWorkflowByID(
			fmt.Sprintf(CustomerWorkflowIdFormat, "guest"),
			SignalCancelAccount, nil)
	}, time.Second*2)

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   1,
		Name:          "Customer",
		Guests:        []string{},
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
			fmt.Println(body)
			return nil
		})

	order := time.Second
	guestId := "guest"
	guestWfId := fmt.Sprintf(CustomerWorkflowIdFormat, guestId)

	// first, invite the guest. This should result in another workflow being started
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalInviteGuest, guestId)
	}, order)
	order += time.Second

	// immediately cancel the guest account. this should keep it queryable, but sets AccountActive -> false
	env.RegisterDelayedCallback(func() {
		fmt.Println("here 0")
		err := env.SignalWorkflowByID(
			guestWfId,
			SignalCancelAccount, nil)
		require.NoError(t, err)
	}, order)
	order += time.Second

	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflowByID(guestWfId, QueryGetStatus)
		require.NoError(t, err)
		var status GetStatusResponse
		err = result.Get(&status)
		require.NoError(t, err)

		require.False(t, status.AccountActive)
	}, order)
	order += time.Second

	// then, try to invite them again. the "guest has already canceled" email should be sent
	env.RegisterDelayedCallback(func() {
		fmt.Println("here2")
		env.SignalWorkflow(SignalInviteGuest, guestId)
	}, order)
	order += time.Second

	// cancel original workflow to not timeout
	env.RegisterDelayedCallback(func() {
		fmt.Println("here3")
		env.SignalWorkflow(SignalCancelAccount, nil)
	}, order)
	order += time.Second

	customer := CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   2,
		Name:          "Customer",
		Guests:        []string{},
		AccountActive: true,
	}
	env.ExecuteWorkflow(CustomerLoyaltyWorkflow, customer)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.NoError(t, env.GetWorkflowResult(nil))

	env.AssertCalled(t, "SendEmail", EmailGuestCanceled)
}

func Test_SendEmailActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	var a *Activities
	env.RegisterActivity(a)

	_, err := env.ExecuteActivity(a.SendEmail, "Hello, World!")
	require.NoError(t, err)
}
