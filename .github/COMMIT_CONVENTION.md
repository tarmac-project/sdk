# Commit Message Convention

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification for commit messages. This leads to more readable messages that are easy to follow when looking through the project history and enables automatic versioning and release notes generation.

## Message Format

Each commit message consists of a **header**, a **body**, and a **footer**.

```
<type>(<scope>): <subject>

<body>

<footer>
```

The **header** is mandatory and must conform to the [Commit Message Header](#commit-message-header) format.

The **body** is optional but encouraged for providing context about the change.

The **footer** is optional and can be used to reference GitHub issues that this commit closes or addresses.

## Commit Message Header

```
<type>(<scope>): <subject>
```

The `<type>` and `<subject>` fields are mandatory, while the `<scope>` field is optional.

### Type

Must be one of the following:

* **feat**: A new feature
* **fix**: A bug fix
* **docs**: Documentation only changes
* **style**: Changes that do not affect the meaning of the code (white-space, formatting, etc)
* **refactor**: A code change that neither fixes a bug nor adds a feature
* **perf**: A code change that improves performance
* **test**: Adding missing or correcting existing tests
* **chore**: Changes to the build process or auxiliary tools and libraries
* **ci**: Changes to CI configuration files and scripts

### Scope

The scope could be anything specifying the place of the commit change.

For example:
* `core`
* `httpclient`
* `kv`
* `log`
* `metrics`
* `sql`
* `function`
* `tests`
* `examples`
* `docs`

### Subject

The subject contains a succinct description of the change:

* Use the imperative, present tense: "change" not "changed" nor "changes"
* Don't capitalize the first letter
* No dot (.) at the end

## Examples

```
feat(core): add new host callback function
```

```
fix(httpclient): correct header parsing in request
```

```
docs: update README with new examples
```

```
chore(ci): update GitHub Actions workflow
```

```
refactor(kv): simplify key management logic
```
