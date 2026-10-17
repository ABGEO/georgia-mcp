---
name: sdd-implement
description: Stage 3 of the spec driven process. Generate the code and the tests from an approved specification. Use after the spec document is approved. Triggers - "implement the spec", "build this feature", "generate the code", "work through the build plan".
---

# Stage 3: Code

Usage: `/sdd-implement <feature key> [step number]`

You generate the code from `docs/specs/features/<key>/spec.md`.

## Read first

1. `docs/specs/features/<key>/spec.md`. Its status must be `approved`.
2. `docs/specs/features/<key>/requirements.md`, for the intent behind each statement.
3. The guidelines of the code you write: `go-style.md`, `testing.md`, `language.md`
   always. `clients.md` for a vendor client. `domain.md` and `package-layout.md` for
   a context. `mcp-tools.md` for a tool. `resilience.md` for a driven adapter.

**Stop if the status of the specification is not `approved`.**

## Steps

1. **Take the next step** of the build plan. Do not start a step before the last one passes.
2. **Write the tests first.** One test for each scenario of the step. The test cites
   its scenario identifier in a comment (`TST-001`). Run the tests. Confirm that they
   fail for the correct reason.
3. **Write the code.** Production code carries no statement identifier (`TRC-007`).
   Write a comment only when the code cannot explain itself (`LNG-011`).
4. **Make the tests pass.** Do not change a test or a scenario to make it pass
   (`TST-003`). A test that looks wrong means the specification is wrong. Stop and
   tell the owner.
5. **Run the checks** of each module that the step touched (`BLD-001`):

   ```
   golangci-lint run ./... && go test -race ./... && go build ./...
   cd pkg/<vendor> && golangci-lint run ./... && go test -race ./...
   ```

   A pass is an exit code. Report the real output. A failure in a module that the
   step did not touch is pre-existing. Name the file.
6. **Mark the step** done in the build plan. Set the `updated` field.
7. **Report** the step, the scenarios that now pass, the real result of each check,
   and each assumption that the specification did not answer.

Do not commit. The owner makes every commit (`GIT-001`).

## When the specification is wrong

Stop. Tell the owner the exact gap. Correct `spec.md` first, then the code.

## Self check before you report

- [ ] Every test cites a scenario identifier. Production code cites none.
- [ ] Every comment is a kind that `LNG-012` allows, with no em dash and no filler word.
- [ ] Every exported identifier has a documentation comment. (`GO-003`)
- [ ] `domain` and `app` import no vendor client, no MCP package, no platform package.
- [ ] A vendor client imports no package of the server. (`CLT-002`)
- [ ] You changed no test and no scenario to make a test pass.
- [ ] The linter, the tests, and the build passed. You have the exit codes.
- [ ] The build plan shows the step as done.
- [ ] You did not commit.
