# gemini

Gemini maintains an in-memory and up to date copy of your Office365 calendar, and exports it in iCalendar format through a secret URL.

## Why?

You work at NAV, and want your calendar on your phone or in other calendar applications.
Unless your device is "compliant", you can forget about it.
Gemini is a workaround for this.

## How does it work?

Go to https://gemini.nais.io and log in with your Microsoft Azure credentials.
Gemini will keep your access and refresh token in a database, and synchronize the calendar at regular intervals.
You will get a personal calendar URL that looks like this:

`https://gemini.nais.io/calendar/<SECRETKEY>`

This URL returns your default calendar in iCalendar format. You can "subscribe" to this URL in your favorite calendar application.

Refreshing your calendar does not trigger calls to Azure. Thus, you can refresh the calendar as often as you like.
