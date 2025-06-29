# Account Logic & Credit Card Workflow

This document explains the business logic behind the `Account` and `Transaction` entities, with a special focus on how the system handles the complex workflow of a Credit Card.

## 1. The Fundamental Concept: The `Account` Entity

In GoFinance Tracker, an `Account` does not just represent a bank account. It is an abstract **"Financial Envelope"** that can represent any place where value is held or owed. Each account belongs to a single user and has a `type` that defines its behavior.

The main account types are:

* **Assets:** Represent money the user **has**.
    * `checking`: Checking Accounts
    * `savings`: Savings Accounts
    * `cash`: Physical cash (wallet)
    * `investment`: Investment accounts (with special balance logic)
* **Liabilities:** Represent money the user **owes**.
    * `credit_card`: Credit Cards

The user's **Net Worth** is the sum of the balances of all their accounts. Since the balances of liability accounts are negative, they naturally subtract from the total.

## 2. Transactions: Moving Value Between Accounts

All movements of value are recorded in the `transactions` table. A transaction is always linked to at least one account.

### Transaction Types

#### a. `expense`

* **Description:** A spending event that decreases your net worth.
* **Example:** Buying a coffee for $10.00 with a debit card.
* **How it's recorded:**
    * `type`: "expense"
    * `amount`: "10.00"
    * `account_id`: The ID of your "Checking Account".
* **Balance Impact:** The balance of the "Checking Account" decreases by $10.00.

#### b. `income`

* **Description:** An inflow of money that increases your net worth.
* **Example:** Receiving a salary of $5,000.00.
* **How it's recorded:**
    * `type`: "income"
    * `amount`: "5000.00"
    * `account_id`: The ID of your "Checking Account".
* **Balance Impact:** The balance of the "Checking Account" increases by $5,000.00.

#### c. `transfer`

* **Description:** The movement of money between **two of the same user's accounts**. A transfer does not change the user's net worth; it only moves money from one "envelope" to another.
* **Example:** Withdrawing $100.00 from an ATM.
* **How it's recorded:**
    * `type`: "transfer"
    * `amount`: "100.00"
    * `account_id` (Source): The ID of your "Checking Account".
    * `destination_account_id` (Destination): The ID of your "Cash Wallet" account.
* **Balance Impact:** The balance of the "Checking Account" decreases by $100.00, and the balance of the "Cash Wallet" increases by $100.00.

## 3. The Complete Credit Card Workflow

This is the most complex and powerful flow in the system. It uses all the concepts above to accurately model financial reality.

### Step 1: Making a Purchase on the Card

When the user spends $150.00 at a restaurant with their credit card:

* **Action:** A transaction of type **`expense`** is created.
* **Associated Account:** The `account_id` is the "Credit Card" account (a liability).
* **Balance Impact:** The credit card's balance, which was, for example, `-500`, now becomes `-650`. The user's debt to the bank has increased, but the cash in their checking account has not yet been touched.

### Step 2: Viewing the Statement (The Smart Logic)

The user wants to see their bill that is due in July.
* **API Action:** The frontend calls `GET /v1/accounts/{card_id}/statement?month=6&year=2025`.
* **Backend Intelligence:**
    1.  The `AccountService` fetches the account details and finds its `statement_closing_day` (e.g., the 25th).
    2.  It calculates the exact statement period: from **May 25th to June 25th**.
    3.  It fetches all `expense` transactions within this date range.
    4.  The API returns the total statement amount (the sum of these transactions) and the list of charges that make it up.

### Step 3: Paying the Bill

This is the moment when money actually leaves the user's asset "pocket". Paying the credit card bill is modeled as a **`transfer`**.

* **Action:** The user registers a payment of $1,500.00 towards their credit card bill.
* **How it's Recorded:**
    * `type`: "transfer"
    * `amount`: "1500.00"
    * **`account_id` (Source):** The ID of the user's "Checking Account".
    * **`destination_account_id` (Destination):** The ID of the "Credit Card" account.
* **Impact on Balances:**
    * **Checking Account (Asset):** The balance decreases by $1,500.00.
    * **Credit Card (Liability):** The balance (the debt), which was, for example, `-2000`, is paid down, moving to `-500` (`-2000 + 1500`).

By modeling the system this way, GoFinance Tracker provides a complete and accurate view of the user's finances, clearly separating money they have from money they owe, and treating debt payments as a logical transfer between these two sides of their financial life.