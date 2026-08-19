# BUG-001 Reproduction

## Symptom

After consecutive failed-password login attempts, the configured lockout threshold is not enforced and rate limiting does not trigger as expected. The failed-login count also does not satisfy the repository contract observed by the regression test.

## Trigger

From the repository root, run:

```bash
go test ./internal/service -count=1 -run '^TestBug001_AuthenticationLockout$'
```

Run this command against the defective `bug1_main` state or another uncorrected BUG-001 candidate.

## Observed Failure

```text
--- FAIL: TestBug001_AuthenticationLockout (0.44s)
    --- FAIL: TestBug001_AuthenticationLockout/rate_limit (0.00s)
        bug001_native_test.go:36: attempt 6: over=false
    --- FAIL: TestBug001_AuthenticationLockout/incremented_repository_count (0.00s)
        bug001_native_test.go:51: call to Query 'SELECT failed_login_count FROM users WHERE id=?' with args [{Name: Ordinal:1 Value:7}], was not expected, next expectation is: ExpectedExec => expecting Exec or ExecContext which:
              - matches sql: 'UPDATE users SET failed_login_count=failed_login_count+1 WHERE id=?'
    --- FAIL: TestBug001_AuthenticationLockout/lock_threshold (0.24s)
        bug001_native_test.go:94: account was not locked at five failures
FAIL
```

## Expected Result

The same regression command exits with code `0` on the repaired candidate, and the rate-limit, repository-count, lock-threshold, correct-password, wrong-password, and successful-reset scenarios pass.
