# Generating the THIRD_PARTY_NOTICES.md file

In the root of the repo, execute the following commands:

```
go get go.elastic.co/go-licence-detector

go list -m -json all | go-licence-detector -rules assets/license/rules.json -noticeTemplate assets/license/THIRD_PARTY_NOTICES.md.tmpl -noticeOut THIRD_PARTY_NOTICES.md -includeIndirect
```
