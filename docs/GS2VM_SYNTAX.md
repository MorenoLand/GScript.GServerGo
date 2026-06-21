# GS2 VM Syntax And Runtime

## Script Splitting

- `//#CLIENTSIDE` splits one weapon or NPC script into server-side and client-side portions.
- The server-side VM runs only the text before `//#CLIENTSIDE`.
- Client bytecode compilation uses the client-side text starting at `//#CLIENTSIDE`.
- Whitespace, blank lines, and indentation are preserved when scripts are saved by NC.

## Event Runtime

- Event handlers execute with `this` bound to the owning script instance.
- Event lookup supports dotted event names such as `HTTPSocket.onBind`.
- Dotted function declarations are translated, so `function HTTPSocket.onBind()` is callable as socket event `HTTPSocket.onBind`.
- Event handler lookup is case-insensitive for `on...` handlers.
- Scheduled event names can be passed as `Kek` or `onKek`; both resolve to `onKek` when that handler exists.
- `onCreated()` runs when a weapon, DB NPC, or level NPC script is applied or loaded.
- `onInitialized()` runs when the NPC-server starts for active server-side scripts.
- `onPlayerLogin()` and `onPlayerLogout()` run for active weapon and DB NPC server-side scripts.
- `onPlayerLogin(pl)` and `onPlayerLogout(pl)` receive the connecting/disconnecting player object; `params[0]` is the same object.
- Single-line functions can omit braces, and top-level comma-separated statements execute in order.

## State

- `this.` values persist across events until the script instance is reloaded.
- Reapplying a weapon, class, or NPC script resets that script instance's `this.` state.
- `temp.` values are event-frame locals and do not survive into the next event.
- `temp.foo = value;` also exposes `foo` as a bare alias in the same event frame.
- Function parameters declared as `temp.name` are locals and also become visible as `name`.
- `params` is the current event argument list.
- `params[0]` and normal indexed access are supported.

## Operators And Translation

- `SPC` concatenates values with a single space.
- `TAB` concatenates values with a tab.
- `NL` concatenates values with a newline.
- `@` concatenates values without adding a separator.
- `@=` appends to strings.
- Bare `TAB` is also available as a tab string.
- Bare `NL` is also available as a newline string.
- `NULL` is JavaScript `null`.
- `public function` is accepted and translated to `function`.
- `enum { A, B, C }` becomes numeric constants starting at `0`.
- `(@funcname)(args...)` dynamically calls a function by name.
- `object.(expr)` dynamically reads or writes a property name.
- `.size()` is translated to `.length`.
- `value in {"a", "b"}` is supported for simple values and inline array literals.

## Arrays

- Inline assignment arrays are supported: `temp.items = {"a", "b"};`
- Inline array arguments are supported: `echo({"a", "b"}[0]);`
- `new[size]` creates a JavaScript array of that size.
- Multi-dimensional `new[x][y]` arrays are supported.
- Array values support `.add(value...)`.
- Array values support `.addarray(array)`.
- Array values support `.insert(index, value)`.
- Array values support `.replace(index, value)`.
- Array values support `.index(value)`.
- Array values support `.indices(value)`.
- Array values support `.delete(index)`.
- Array values support `.remove(value)`.
- Array values support `.clear()`.
- Array values support `.sortascending()`.
- Array values support `.sortdescending()`.
- Array values support `.sortbyvalue(name, type, ascending)`.
- Array values support `.insertarray(index, array)`.
- Array values support `.subarray(start, length)`.
- Array values support `.loadlines(filename)`.
- Array values support `.savelines(filename, mode)`.

## Loops

- Classic `for (...)` loops are supported.
- `For (...)` is normalized to `for (...)`.
- `do { ... } while (condition);` loops are supported.
- `for (temp.item : list)` is supported.
- `for (temp.item in list)` is supported.
- Foreach loops with `temp.item` expose both `temp.item` and bare `item` inside the loop body.
- `maxlooplimit` defaults to `10000` loop ticks and can be raised per script before a heavy loop.

## Strings

- String values support `.substring(start)` and `.substring(start, length)` using GS2 length semantics.
- String values support `.tokenize(delimiter)`.
- String values support `.pos(needle)`.
- String values support `.starts(prefix)`.
- String values support `.startswith(prefix)`.
- String values support `.ends(suffix)`.
- String values support `.endswith(suffix)`.
- String values support `.trim()`.
- String values support `.lower()`.
- String values support `.upper()`.
- String methods work on variables, string literals, and parenthesized string expressions.
- String values support `.loadstring(filename)`.
- String values support `.savestring(filename, mode)`.
