# ♊ Gemini

Gemini is a Office365 to iCalendar bridge.

It maintains an in-memory and up-to-date copy of your Office365 calendar,
strips it of attachments, and exposes it in iCalendar format through a secret
URL that can be reached on the Internet.

## Why?

You work at an organization that uses Office365 e-mail, and want to see your
work calendar alongside your other calendars.  Unless your device is
"compliant", you can forget about it.
Gemini is the solution.

## Features

The following information is retained:

* Read-only copy of your Office365 calendar.
* Meeting title, description, participants.
* Recurring meetings work as expected.
* Alerts/notifications.
* Timezone information.

Gemini removes:

* Any attachments.
* Microsoft cruft.
* HTML formatted text.

Good to know:

* You can share your calendar with others through a separate public calendar URL that gives only "busy/available" status.
* Works on all _iCalendar_ compatible readers, such as Google Calendar, Apple Calendar, or literally any other tool.
* No OAuth2 or Exchange protocol support needed in your client. Your private URL is your secret key, don't share it.
* This calendar is read-only. Unfortunately, this is a design constraint with _iCalendar_ format and cannot be added.
* If your calendar cannot be retrieved, you will be notified through your calendar integration URL; the current week will be filled with meetings exposing the error message.

## Usage

Go to https://gemini.external.prod-gcp.nav.cloud.nais.io and log in with your Microsoft Azure credentials
to activate calendar syncing.

Gemini will synchronize the calendar every thirty minutes.

You get a personal calendar URL that looks like this:

`https://gemini.external.prod-gcp.nav.cloud.nais.io/calendar/<SECRETKEY>`

This URL returns your default calendar in iCalendar format. You can _subscribe_ to this URL in your favorite calendar application.

Refresh your calendar as often as you like, Gemini doesn't care.

## Operation

You need an _Azure App Registration_ with a client ID and secret.

Additionally, you need to assign the following permissions from Microsoft Graph API:

* `Calendars.Read` - read user calendars
* `GroupMember.Read.All` - read group memberships
* `User.Read` - sign in and read user profile
* `openid` - sign users in
* `Calendars.ReadWrite` - full access to user calendars, may or may not be needed, your mileage may vary.

There is a Prometheus endpoint at `/metrics` for scraping metrics.
You may use the [sample dashboard](grafana.json) as a starting point for visualizing in Grafana.

## Authors

Written and maintained by Kim Tore Jensen <<kim.tore.jensen@nav.no>>.
