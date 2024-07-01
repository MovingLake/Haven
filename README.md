# haven
Haven is an OSS tool to run continuous data QA.

Currently it works only on JSON payloads. Given a stream of JSON payloads for a resource (such as an API endpoint) Haven will compute it's ongoing schema and emit notifications to the URL provided whenever a change is detected.
