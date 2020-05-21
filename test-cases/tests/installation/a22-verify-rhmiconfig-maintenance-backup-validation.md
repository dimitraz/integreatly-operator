---
targets:
- 2.3.0
estimate: 30m
---

# A22 - Verify RHMIConfig maintenance and backup validation
This test is to verify that the RHMIConfig validation webhook for maintenance and backup values works as expected.

The expected value formats are:
```yaml
spec:
  backup: 
    applyOn : "HH:mm"
  maintenance: 
     applyFrom : "DDD HH:mm"
```

1. Add new values in correct format to ensure the validation works with no error. We require the expected values outlined above. We also expect these values to be parsed as a 1 hour window, which should not overlap 

2. Add poorly formatted values for the `backup.applyOn` and `maintenance.applyFrom` fields, e.g: `"wefwef:wfwefwef", 12:111, sqef 12:13, 42:23` etc and ensure that the validation webhook does not allow these changes to be made

3. Add overlapping values for these time windows, e.g:
```yaml
spec:
  backup: 
    applyOn : "22:01"
  maintenance: 
     applyFrom : "Mon 22:20"
```
Ensure that the validation webhook does not allow these values. 
