# ♊ Gemini

Gemini maintains an in-memory and up-to-date copy of your Office365 calendar.
A stripped-down version is exported in iCalendar format through a secret URL that is available on the Internet.

## Why?

You work at NAV, and want your calendar on your phone or in other calendar applications.
Unless your device is "compliant", you can forget about it.
Gemini is a workaround for this.

## How does it work?

Go to https://gemini.external.prod-gcp.nav.cloud.nais.io and log in with your Microsoft Azure credentials
to activate calendar syncing.

Gemini will synchronize the calendar at regular intervals.

You will get a personal calendar URL that looks like this:

`https://gemini.external.prod-gcp.nav.cloud.nais.io/calendar/<SECRETKEY>`

This URL returns your default calendar in iCalendar format. You can "subscribe" to this URL in your favorite calendar application.

You can refresh your calendar as often as you like. Internally, the calendar is synced every hour.
