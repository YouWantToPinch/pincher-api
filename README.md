# Pincher API
<p>
  <img width="468" height="275" alt="pincher-api-logo-1" src="https://github.com/user-attachments/assets/ed3b603e-27de-4af5-b4f1-269d79022d41" />
</p>

## About the API
Pincher is a webserver that hosts manual budgeting tools designed for meticulous manual user control in a zero-based approach to personal finance.
As a Go-based webserver, Pincher was designed for the purpose of providing a FOSS solution to this budgeting philosophy that can also provide a REST API 
that users can leverage for custom tooling, as well as the benefit of concurrency.

The Pincher API exposes endpoints which can be used to create and manage resources such as:
- Users
- Budgets
- User Budget Memberships
- Categories
- Category Groups
- Accounts
- Payees
- Transactions (Deposits, Withdrawals, Transfers)
- Money Assignments by Month

All of these resources are kept within a Postgres database.

## Resources
Official documentation is currently being written.

[About zero-based budgeting](https://en.wikipedia.org/wiki/Zero-based_budgeting) (also known as [envelope budgeting](https://en.wikipedia.org/wiki/Envelope_system))

