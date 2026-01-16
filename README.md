# Pincher API
<p>
  <img width="468" height="275" alt="pincher-api-logo-1" src="https://github.com/user-attachments/assets/ed3b603e-27de-4af5-b4f1-269d79022d41" />
</p>

A web server exposing a REST API used to manage budgetary resources that represent the components necessitated by an envelope framework of personal finance.


## Why Pincher?

The very idea of efficiently budgeting any limited resource—or any amount of that resource, to ensure that it does not dwindle—necessitates that the cost of doing so is not bound to the resource itself. Or, to say it more plainly: ***budgeting should be monetarily free.***

Beyond your personal task of acquiring a computer to run the software on, Pincher is here for you, and it's here *for free* and *for life.* Why should you ever be priced out of the opportunity to use the tools you need to **SAVE**?

Still, while other FOSS solutions in the budgeting space are gaining a respectable traction, why Pincher?

Pincher's entire goal is to provide you with *options.*

While others FOSS solutions to envelope budgeting do exist, they often feature a JavaScript back end infrastructure with no REST API. Pincher offers you endpoints that you can use to write custom tooling for your budgetary needs in whatever language you wish, and, being Go-based, Pincher is equipped with the benefit of leveraging Go's concurrency primitives.

Furthermore, Pincher doesn't strive *just* to provide tooling for individuals; it is made with families and small businesses in mind. Maybe your spouse isn't interested in managing the budget, and you've taken on that task; all the same, they can be assigned the VIEWER role so that they might be able to check budget balances at will to stay up to date. Or, maybe you would like to be on the same page with your employees about the nature of the charges that come through on company cards. Give them the CONTRIBUTOR role, so that they can log transactions to the budget!

Pincher is what you make of it!

## Get Started

The recommended method for setup is to run Pincher in a Docker container, with its environment variables set as needed to build a URL which points to a Postgres database.

As of now, no official pincher-api images are released on any regular basis, so for now, clone the repository to build it locally:

```
git clone https://github.com/YouWantToPinch/pincher-api.git
```

A `docker-compose.yml` file is already provided in the repository as a template.
Before building, it is recommended that you generate a better secret for your JSON Web Token authentication. You can do so with `openssl`:

```
openssl rand -base64 64
```

In the `docker-compose.yml`, set any other environment variables you would like. It is recommended that `MIGRATE_ON_START` is set to `true`, so that should you ever provide a new binary, its embedded SQL migrations will be applied after the update.

Run the `./scripts/start_pincher_server.sh` script. It will build the `pincher-api` image, as well as a Postgres image that hosts your database.

You're now free to interact with the API!

If you need a client to get started, consider the [official Pincher CLI](https://github.com/YouWantToPinch/pincher-cli.git).

## Usage

The Pincher API exposes endpoints which can be used to create and manage items such as:
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

*Official documentation is currently being written.*

Until documentation is complete, for reference, all endpoints are matched directly to their function handlers within `./internal/api/api.go`. There, you will also see one of the following roles listed:

- ADMIN
- MANAGER
- CONTRIBUTOR
- VIEWER

These are **user member roles.** That is, these endpoints will all inherently expect a JWT in the header of the requests sent to them, which can be acquired through the user login endpoint.

The budget MANAGER role carries almost all of the same privileges as the ADMIN role, save just for budget deletion.

## Resources

[Read about zero-based budgeting](https://en.wikipedia.org/wiki/Zero-based_budgeting) (also known as [envelope budgeting](https://en.wikipedia.org/wiki/Envelope_system))

