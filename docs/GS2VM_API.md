# GS2 VM Globals And API

## Core Globals

- `name`
- `params`
- `temp`
- `this`
- `player`
- `client`
- `clientr`
- `chat`
- `server`
- `serverr`
- `serveroptions`
- `allplayers`
- `weapons`
- `screenwidth`
- `screenheight`
- `TAB`
- `NL`
- `NULL`

`TAB` and `NL` also work as GS2 concat tokens between expressions.

## Core Functions

- `echo(value...)`
- `int(value)`
- `random(min, max)`
- `char(code)`
- `strlen(value)`
- `isObject(value)`
- `replacetext(value, search, replacement)`
- `toJson(value)`
- `base64encode(value)`
- `base64decode(value)`
- `openurl(value)`
- `sleep(value)`

## Class And Scheduling Functions

- `loadclass(name)` is accepted as a no-op runtime call.
- `join(name)` is accepted as a no-op runtime call after server-side class expansion has already happened.
- `leave(name)` is accepted as a no-op runtime call.
- `scheduleevent(delay, event)`
- `scheduleEvent(delay, event)`
- `this.scheduleevent(delay, event)`
- `this.scheduleEvent(delay, event)`
- `this.join(name)`
- `this.leave(name)`

## Player Functions

- `findplayer(value)`
- `setlevel(level)`
- `setlevel2(level, x, y)`
- `addweapon(name)`
- `removeweapon(name)`

Bare `setlevel`, `setlevel2`, `addweapon`, and `removeweapon` target the player that triggered the current server-side event.

## Player Objects

Player objects are returned by `findplayer()` and appear in `player` and `allplayers`.

Supported fields:

- `account`
- `nick`
- `nickname`
- `level`
- `client`
- `clientr`

Supported methods:

- `sendpm(message)`
- `sendplayer(message)`
- `setlevel(level)`
- `setlevel2(level, x, y)`
- `addweapon(name)`
- `removeweapon(name)`

`sendplayer()` is treated as a compatible alias for `sendpm()`.

Supported lookup keys:

- account name
- current nickname
- PCID-style guest identity such as `pc:763`

Supported call forms:

- `temp.pl = findplayer("moondeath"); temp.pl.sendpm("hey");`
- `temp.pl = findplayer("moondeath"); pl.sendpm("hey");`
- `findplayer("moondeath").sendpm("hey");`

## Player Flags

- `player.client.flag`
- `player.clientr.flag`
- `client.flag`
- `clientr.flag`

Assignments to `client.` and `clientr.` update the owning player's flags and queue matching flag updates for the gserver bridge.

## Server Flags

- `server.flag`
- `serverr.flag`

Assignments update server flags and deletes remove server flags.

Examples:

- `server.foo = "bar";`
- `serverr.secret = true;`
- `delete server.oldflag;`

## Server Options

- `serveroptions.optionname`

Server options are exposed read-only.

Comma-separated option values are exposed as list-like values for indexed access.

Example:

- `echo(serveroptions.staff[1]);`

## Player And Weapon Lists

- `allplayers` is an array of player objects visible to the VM.
- `weapons` is an array of weapon objects.

Weapon object fields:

- `name`
- `image`

## Triggers

- `triggerclient(type, target, args...)`

The VM queues a client trigger result for the gserver bridge.

Client `triggerServer("gui", name, args...)` and `triggerServer("weapon", name, args...)` arrive through triggeraction handling and dispatch to matching server-side script events.

## Drawing Functions

- `showimg(index, image, x, y)`
- `findimg(index)`
- `getimgwidth(image)`
- `getimgheight(image)`

Image objects currently expose:

- `index`
- `image`
- `x`
- `y`
- `rotation`

## File Functions

- `loadstring(filename)`
- `loadlines(filename)`
- `savestring(filename, value, mode)`
- `savelines(filename, lines, mode)`
- `findfiles(pattern, recursive)`

Save mode accepts overwrite by default and append when mode is `1`, `true`, or `append`.

File operations are rooted to the configured VM file root and reject absolute paths or paths escaping that root.

## NPC Functions

- `setshape(shapeType, width, height)`
- `setshape2(width, height, tileTypes)`
- `warpto(level, x, y)`

These only emit NPC actions when the VM run has an NPC ID.

## TSocket Functions And Objects

- `new TSocket(name)`
- socket `.bind(port, ssl)`
- socket `.send(data)`
- socket `.close()`
- socket `.destroy()`
- socket `.join(name)`
- bare `send(data)` in socket events
- bare `close()` in socket events

Socket object fields:

- `name`
- `objecttype`
- `address`
- `error`
- `ipaddress`
- `isconnected`
- `port`
- `parent`
- `data`
- `packagedelimiter`
- `enablessl`

Socket event globals:

- `outdatalength`
- `isconnected`

Supported socket events are routed by dotted function names, including:

- `SocketName.onBind`
- `SocketName.onBindFailed`
- `SocketName.onNewClient`
- `SocketName.onReceiveDataPackage`
- `SocketName.onDisconnect`

## Compatibility No-Ops

These functions exist so scripts can run while host behavior is implemented elsewhere or intentionally ignored:

- `loadclass`
- `join`
- `leave`
- `openurl`
- `Adventure_setAllowedPortsBind`
- `sleep`
