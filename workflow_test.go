package starter

import (
	"fmt"
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

func Test_InviteGuest(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	// first, invite the guest. This should result in another workflow being starter
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

func Test_SendEmailActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	var a *Activities
	env.RegisterActivity(a)

	_, err := env.ExecuteActivity(a.SendEmail, "Hello, World!")
	require.NoError(t, err)
}
