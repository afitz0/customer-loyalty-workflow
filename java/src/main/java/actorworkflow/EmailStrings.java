package actorworkflow;

public final class EmailStrings {
    private EmailStrings() {}

    static final String EMAIL_WELCOME = "Welcome to our loyalty program! You're starting out at '%s' status.";
    static final String EMAIL_GUEST_CANCELED = "Sorry, your guest has already canceled their account.";
    static final String EMAIL_GUEST_INVITED = "Congratulations! Your guest has been invited!";
    static final String EMAIL_INSUFFICIENT_POINTS = "Sorry, you need to earn more points to invite more guests!";
    static final String EMAIL_PROMOTED = "Congratulations! You've been promoted to '%s' status!";
    static final String EMAIL_CANCEL_ACCOUNT = "Sorry to see you go!";
    static final String EMAIL_DEMOTED = "Unfortunately, you've lost enough points to bump you down to '{}' status. 😞";
    static final String EMAIL_GUEST_MIN_STATUS = "Your guest already has an account, but we've made sure they're at least '%s' status!";
}
