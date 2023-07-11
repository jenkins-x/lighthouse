## Reuse steps in a task mixed with custom tasks 

This example shows how we can use `image: uses:sourceURI` and a `name: mystep` to include individual the steps in task and mix them with custom local steps (which don't use `uses:`)

This example uses the concise syntax; where empty `image:` values get inherited from the `stepTemplate.image`
