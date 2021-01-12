# ringtest from Programming Erlang First Edition Chapter 8 Problem 2

Done in erlang and golang.

The ring creation times in erlang are murky at best. Read the code
and judge yourself.

# Observations

In erlang it is easier to mentally grasp "processes" in my opinion
as a single process has only one mailbox represented by the pid.

In go it is a bit more cumbersome to manage those inboxes yourself.

There are several possibilities, I opted for creating an inbox for
each message type. So a goroutine has multiple mailboxes in erlang
speak.

Message passing seems to be twice as fast in go.

# How to run the programs

Install go and install erlang and look in the `run` file
which commands to issue.

Or have [nix](https://nixos.org/) installed and just exec the run file.
