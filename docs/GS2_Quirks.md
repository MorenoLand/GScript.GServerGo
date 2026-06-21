# GS2 Quirks And Runtime Behavior

This is a working notes file for `gserver-go`, `npcserver.go`, and the native Go GS2 VM. Keep this factual and update it when behavior is verified against real clients, C++ GServer, or the C# engines.

## Variable Scope

- `temp.` variables are event-frame locals. They exist only for the current function/event call and must not survive into the next event.
- `temp.foo = value;` also exposes `foo` as a bare local alias inside that same event frame. These should both work:
  - `temp.foo = findplayer("moondeath"); temp.foo.sendpm("kek");`
  - `temp.foo = findplayer("moondeath"); foo.sendpm("kek");`
- `this.` variables are script-instance state. For weapons, they persist across server-side events until the weapon script is reapplied/recompiled.
- Reapplying a weapon/class/NPC script should reset its `this.` state because it is effectively a new runtime instance.
- `getstringkeys()` is for persistent variable prefixes such as `this.modtime_`, not `temp.` locals.

## Global And Object Access

- `player` is the current server-side player context when the event is player-triggered.
- `player.client.*` and bare `client.*` both refer to client flags on the current player.
- `player.clientr.*` and bare `clientr.*` both refer to clientr flags on the current player.
- `server.*` and `serverr.*` come from server flags.
- `serveroptions.*` comes from server options.
- Comma-separated server option values should behave like arrays for indexed access, e.g. `serveroptions.staff[1]`.

## Player Lookup And PMs

- `findplayer(str)` returns a player object or `null`.
- Matching should accept account, nickname, and PCID-style guest account names such as `pc:763`.
- `sendpm(str)` and `sendplayer(str)` on a player object send a PM from NPC-Server.
- These forms should work:
  - `temp.pl = findplayer("moondeath"); temp.pl.sendpm("hey");`
  - `temp.pl = findplayer("moondeath"); pl.sendpm("hey");`
  - `findplayer("moondeath").sendpm("hey");`
- Calling a method on `null` should report a clean compiler/runtime-style error to NC, not dump raw VM exception text.

## Operators And Syntax Notes

- `SPC` concatenates with one space.
- `@` concatenates without a space.
- `//#CLIENTSIDE` splits server-side and client-side code. Server-side compilation/runtime should only run the portion before this marker.
- Blank lines or code placement around `//#CLIENTSIDE` can affect client behavior, so preserve script text formatting exactly when saving through NC.

## Triggers

- `triggerserver(type, name, args...)` from clientside routes to server-side script handling for the target weapon/GUI/NPC.
- Server-side weapon trigger handlers should resolve case-insensitive `onActionServerside` / `onActionServerSide` style names.
- Trigger parameters are exposed through `params`, and indexed access such as `params[0]` must work.
- `triggerclient(type, target, args...)` sends a client-side trigger back to the player for the named GUI/weapon.
- Duplicate client trigger packets can happen from some clients; server-side handling should avoid running the same trigger twice for the same client action.

## Error Output

- NC-facing GS2 runtime errors should use the same style family as compiler feedback.
- Preferred format:
  - `Compiler error for Weapon -gr_movement:`
  - `error: Cannot read property 'sendpm' of undefined or null at line 5`
- Do not print full bytecode blobs, large hex dumps, or raw VM exception payloads unless packet/debug server options explicitly ask for that level of logging.

## Current Go VM Coverage

- Implemented: `echo`, `params`, `temp.` locals with bare aliases, `this.` runtime state, `player`, `client`, `clientr`, `server`, `serverr`, `serveroptions`, `findplayer`, `sendpm`, `sendplayer`, and `triggerclient`.
- Pending/verify against references: broader GS2 object model, `@` edge cases, array/object literal parity, server-side events beyond weapons, class/NPC-db state, timers, waits, scheduleevent, and full trigger target routing.
