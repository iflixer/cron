CRON service

tables in DB
flix_cron
flix_cron_log

takes the tasks from DB and loads the targets urls according to cron expressions

updates the tasks on the fly if db changed

saves response code and text to log