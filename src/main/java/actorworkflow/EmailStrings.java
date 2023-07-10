package actorworkflow;

public interface EmailStrings {
    static public String EMAIL_WELCOME = "Welcome to our loyalty program! You're starting out at '%s' status.";
    static public String EMAIL_GUEST_CANCELED = "Sorry, your guest has already canceled their account.";
    static public String EMAIL_GUEST_INVITED = "Congratulations! Your guest has been invited!";
    static public String EMAIL_INSUFFICIENT_POINTS = "Sorry, you need to earn more points to invite more guests!";
    static public String EMAIL_PROMOTED = "Congratulations! You've been promoted to '%s' status!";
    static public String EMAIL_CANCEL_ACCOUNT = "Sorry to see you go!";
    static public String EMAIL_DEMOTED = "Unfortunately, you've lost enough points to bump you down to '{}' status. 😞";
    static public String EMAIL_GUEST_MIN_STATUS = "Your guest already has an account, but we've made sure they're at least '%s' status!";
}
