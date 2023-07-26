from temporalio import activity


@activity.defn
async def send_email(body: str) -> None:
    activity.logger.info("Sending email with contents %s" % body)


@activity.defn
async def start_guest_workflow(guest_id: str) -> bool:
    activity.logger.info("Starting guest workflow with ID %s" % guest_id)
    return True
