# My Hours

This is a simple leisure project to build a small CLI for tracking daily hours at work.

**Key requirements**:
* must be able to start and stop tracking freely
* able to get a summary of time spent per day, week, and month
* data must persist and can be transferred between computers if needed.

## Current state

* Ability to track time is there
* Supports three categories of time (uncategorized, personal, and work)
* Timer is preserved if program is closed (restored on startup)
* Supports weekly, monthly and yearly reports
  * reports can be fetched independently per category.
* Support importing data from a text file.

## Roadmap

Everything is done on best effort, when-I-feel-like-it basis. With that said, some things that could be taken care of in the near future:
* Cleanup the reporting models. It's too complicated.
* Cleanup the signaling inside the app, it's a mixture of key events and internal messages. Horrible.
* Fixing broken records (if you accidentally leave the timer running over night for example) needs to be done via SQL. This could be fixed with as simple CLI option for example, similar to import.