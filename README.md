# My Hours

[![go build and test](https://github.com/msepp/myhours/actions/workflows/go-test.yml/badge.svg)](https://github.com/msepp/myhours/actions/workflows/go-test.yml)
[![golangci-lint](https://github.com/msepp/myhours/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/msepp/myhours/actions/workflows/golangci-lint.yml)

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
* Support adding custom categories
  * Category selecting needs to be better if we there's more than 3, or custom amount.
* Notes are missing. Would be nice to be able to record for example issue ids per record.
* No way to amend records currently, except going into the database directly.