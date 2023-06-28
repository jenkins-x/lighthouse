## Reuse steps in a task 

This example shows how we can use `image: uses:sourceURI` and a `name: mystep` to include individual the steps in task - and using the `@versionStream` which is resolved with the version stream to find the actual git tag to use.