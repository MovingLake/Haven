# Haven
Haven is an OSS tool to run continuous data QA.

## Requirements

Haven uses a postgres DB to store resources, schemas and their versions. You can load the postgres address through the `.env` file or through command line flags (see the `.env` file for specifics).

Haven uses gORM to manage the DB. It uses connection pooling by default.

## DB Layout

Haven uses three tables:

1. Resources: Table which tracks the schema for a JSON resource. Example:
```
Resource {
    Name: "/api/v1/payments"
    Schema: "my stringified json schema"
    Version: 1
}
```
2. ResourceVersions: Tracks the resource versions across time.
```
ResourceVersions {
    Version: 1
	ResourceID: 1
	ReferencePayloadsID: 0
	OldSchema: "My old json schema"
	NewSchema: "This version's schema"
}
```
3. ReferencePayloads: For auto-generated schemas, this table stores the payloads that generated each version.
```
ReferencePayloads {
	ResourceID: 2
	Payload: "My JSON payload"
}
```

## Usage

There's two main ways to use Haven:
1. Auto-generated schema
2. Manually set schema

### Auto-generated schema

`/api/v1/add_payload` is your friend here. Any JSONs sent to this endpoint will automatically create a Resource in Haven and will get an auto-generated schema. 

Once the schema is to your satisfaction you can use `/api/v1/validate_payload` to validate whether a payload matches your schema.

### Manually set schema

You can also just manually set the schema `/api/v1/set_schema` and then use `/api/v1/validate_payload` to test payloads against the saved schema.

## Testing

Haven uses unit and functional tests. Unit tests do not have any external dependency and test the code in isolation. Functional tests need a postgres DB to run named `haventest` running in localhost.