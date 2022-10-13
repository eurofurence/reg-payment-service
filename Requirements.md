## Implementation related questions

---

In the transactions table, the transaction ID will be used to determine if a row needs to be updated.
Should it be a unique index?

In the transaction log table we will only append transactions after creation, the transaction ID won't be unique here.

---

Is there a transaction type declined?

---

What actually is the meaning of amount in a transaction?

---
