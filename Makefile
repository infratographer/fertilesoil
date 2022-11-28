.PHONY: generate
generate:
	go generate ./...
	sed -i 's/AddChildrenIDs/AddChildIDs/' ent/proto/entpb/entpb_directory_service.go
