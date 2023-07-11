package actorworkflow;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class CustomerLoyaltyActivitiesImpl implements CustomerLoyaltyActivities {
    private final boolean HAS_BUG = false;

    private static final Logger logger = LoggerFactory.getLogger(CustomerLoyaltyActivitiesImpl.class);

    @Override
    public void sendEmail(String body) {
        logger.info("Sending email: '{}'.", body);
        /// Blocking REST
    }

}
