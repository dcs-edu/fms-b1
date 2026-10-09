# School Fees & Management Backend

A Go backend for running a small school: staff and parent accounts, student records, fee bills, and
payments. Parents (and the bursar) pay fees online through [Paystack](https://paystack.com); only the
admin or principal can record cash.

It was built to stop a real problem: small amounts of fee money going missing and adding up over a
school year. So every payment is traceable: who started it, how it was paid, and whether money
actually arrived.

## How money is handled

These rules are what the whole payments design is built around:

- **Every payment has a status.** `pending` (the payer was sent to Paystack), `completed` (money
  received), or `failed` (Paystack reported the charge failed). Failed attempts are kept as evidence.
- **Only `completed` payments count as money.** Every balance and total filters on it.
- **Every payment records who started it** (`initiated_by`): the staff member for cash, the parent or
  bursar for online payments.
- **The bursar cannot record cash.** They pay online like a parent, so every payment they make leaves
  a Paystack record. Only the admin and principal can record a cash payment.
- **Nobody can overpay a bill.** Cash payments check the balance inside a database transaction;
  online payments check it before sending the payer to Paystack.
- **Paystack is the source of truth for online payments.** A payment becomes `completed` when the
  Paystack webhook arrives (checked with an HMAC signature and against the expected amount), or when
  someone asks for its status and the server verifies it with Paystack. That second path catches
  webhooks that never arrived.

## Tech stack

- **Go** with [gorilla/mux](https://github.com/gorilla/mux) for routing
- **PostgreSQL 17** through [sqlx](https://github.com/jmoiron/sqlx) and the [pgx](https://github.com/jackc/pgx) driver
- **[golang-migrate](https://github.com/golang-migrate/migrate)** for schema migrations
- **JWT** auth ([golang-jwt](https://github.com/golang-jwt/jwt)) with bcrypt-hashed passwords
- **Paystack** through a small hand-written client (no SDK), in `internal/paystack`
- **Docker Compose** for the local database

## Project layout

```
cmd/api/              entry point: config, wiring, routes
internal/
  auth/               roles and JWT creation
  middleware/         logging and JWT checking
  handlers/           HTTP handlers (one file per area: users, students, payments, ...)
  repository/         all SQL; maps Postgres errors to the app's own errors
  models/             structs shared by handlers and repository
  paystack/           Paystack API client, webhook signature check, pesewa conversion
  db/                 database connection
pkg/                  small pure helpers (student IDs, date ranges, graduation year), with tests
migrations/           numbered up/down SQL migrations
```

## Getting started

### Requirements

- Go 1.26+
- Docker with Compose
- The `migrate` CLI from [golang-migrate](https://github.com/golang-migrate/migrate)
- A Paystack account (test keys are enough)

### 1. Configure

Create a `.env` file in the project root (it is gitignored):

```env
DB_USER=school
DB_PWD=change-me
DB_NAME=school
DB_URL=postgres://school:change-me@localhost:5432/school?sslmode=disable
PORT=":8080"
JWTSECRET=a-long-random-string
PAYSTACK_SECRET_KEY=sk_test_xxxxxxxxxxxxxxxx
```

| Variable | What it is |
|---|---|
| `DB_USER`, `DB_PWD`, `DB_NAME` | Used by Docker Compose to create the Postgres database |
| `DB_URL` | Connection string the server and `migrate` use |
| `PORT` | Address to listen on, with the colon (`":8080"`) |
| `JWTSECRET` | Secret for signing login tokens. Make it long and random |
| `PAYSTACK_SECRET_KEY` | From the Paystack dashboard, **Settings → API Keys & Webhooks** |

The server refuses to start if `DB_URL`, `PORT` or `PAYSTACK_SECRET_KEY` is missing.

### 2. Run

```sh
make up            # start Postgres in Docker
make migrate-up    # apply all migrations
make run           # start the API on $PORT
```

Other targets: `make down` (stop Postgres), `make migrate-down` (roll back one migration),
`make dbversion` (show the current migration).

### 3. Create the first admin

Everyone who registers starts as a `parent`. Promote your own account to admin once, directly in
the database:

```sql
UPDATE users SET role = 'admin' WHERE email = 'you@example.com';
```

Log in again afterwards: the role is stored in the token when you log in. From then on, the admin
can change other people's roles through `PATCH /admin/users/role`.

### 4. Receive Paystack webhooks locally

Paystack has to reach your machine to deliver webhooks. While developing, a
[Cloudflare quick tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/do-more-with-tunnels/trycloudflare/)
does this without an account:

```sh
cloudflared tunnel --url http://localhost:8080
```

It prints a `https://<random-words>.trycloudflare.com` address. Set the **Test Webhook URL** in the
Paystack dashboard to `https://<random-words>.trycloudflare.com/webhooks/paystack`. A quick tunnel gets a
new address each time it starts, so update the dashboard whenever you restart it.

## API

All endpoints except `/auth/*` and `/webhooks/paystack` need an `Authorization: Bearer <token>` header,
using the token from `/auth/login`. Students are identified by their short ID (e.g. `33-0002`), the one
printed on their card. Money is sent as a string (`"150.00"`) so it never passes through a float.

**Roles:** `admin`, `principal`, `bursar`, `teacher`, `parent`.

### Accounts

| Method | Path | Who | Body / notes |
|---|---|---|---|
| POST | `/auth/register` | anyone | `{"name", "email", "phone", "password"}`; new accounts are parents |
| POST | `/auth/login` | anyone | `{"email", "password"}` → `{"token"}` |
| PATCH | `/account/password` | any logged-in user | `{"old_password", "new_password"}` |
| PATCH | `/admin/users/role` | admin | `{"email", "role"}`; you can't change your own role |

### Students and parents

| Method | Path | Who | Body / notes |
|---|---|---|---|
| POST | `/new/student` | admin, principal | student details; returns the new student ID |
| POST | `/new/book` | admin, principal | inventory |
| POST | `/parents/links` | admin, principal | `{"parent_email", "student_id"}`: staff confirm who a parent is |
| GET | `/me/children` | parent | the students linked to you |

### Bills

A *utility* is something the school charges for (tuition, feeding, ...). A *price* for a utility in a
given semester is a **bill** that every student owes.

| Method | Path | Who | Body / notes |
|---|---|---|---|
| POST | `/utilities` | admin, principal, bursar | `{"util_name": "Tuition"}` |
| POST | `/utilities/{util_id}/prices` | admin, principal, bursar | `{"sem": "2026-T1", "amount": "500.00"}` → returns the `bill_id` |

### Payments

| Method | Path | Who | Body / notes |
|---|---|---|---|
| POST | `/payments/online` | parent (own children), bursar | `{"student_id", "bill_id", "amount"}` → `authorization_url` to send the payer to |
| GET | `/payments/online/{reference}` | the parent, staff | Current status; asks Paystack if not completed yet |
| POST | `/payments` | admin, principal | Record a **cash** payment: `{"student_id", "bill_id", "amount"}` |
| GET | `/payments?range=...` | admin, principal, bursar | Every payment attempt in a range |
| GET | `/payments/students/{student_id}?range=...` | parent (own children), staff | One student's payments |
| POST | `/webhooks/paystack` | Paystack | Checked with the `x-paystack-signature` header |

`range` is one of `today`, `this_week`, `this_month`, `three_months_to_date`. Parents never see
`initiated_by` (which staff member handled a payment is internal).

### An online payment, start to finish

1. The parent calls `POST /payments/online`. The server checks the parent is linked to the student and
   that the amount isn't more than what's left on the bill, saves a `pending` payment with a fresh
   reference, and asks Paystack for a checkout page.
2. The parent pays on that page.
3. Paystack calls `POST /webhooks/paystack`. The server checks the signature and the amount, then marks
   the payment `completed`. A repeated webhook changes nothing.
4. If the webhook never arrives, `GET /payments/online/{reference}` asks Paystack directly and updates
   the payment.

## Tests

```sh
go test ./...
```

Unit tests currently cover the helpers in `pkg/`. Handler tests are planned.

## Status

Work in progress: the backend works end to end with Paystack test keys, and a frontend is next.
