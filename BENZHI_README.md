# Evaluation Guide

- Repository: zhangkui/go-fleet-maintenance
- Branch: test_model_fix3

## Public verification

```bash
go test ./internal/service -count=1 -run '^TestBug003_PasswordChangeRevokesUserSessions$'
```
