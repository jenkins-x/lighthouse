## Reuse steps in a task mixed with custom tasks 

This example shows how we can use `image: uses:sourceURI` and a `name: mystep` to include individual the steps in task and mix them with custom local steps (which don't use `uses:`)

This example uses the verbose syntax where each step always has an image (rather than inheriting from `stepTemplate.image`)