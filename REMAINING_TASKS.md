# Remaining Implementation Tasks

## Feature 5: Trait Default Methods

- [x] **5a.** Store trait method bodies: in `registerTrait`, if a method `FnDecl` has a `Body`, save it in a new map `traitDefaultBodies[traitName+"."+methodName] = FnDecl`
- [x] **5b.** In `registerImpl`, track which methods each impl overrides: build a set of overridden method names per impl
- [x] **5c.** After `registerImpl`, for each trait impl where a trait method is NOT overridden, clone the default body `FnDecl` with the target type substituted for `Self`, and register it as a concrete method
- [x] **5d.** In the driver, emit the cloned default method bodies as top-level functions (same as regular impl methods) — done by design: cloned FnDecl appended to mod.File.Items, driver picks it up automatically
- [x] **5e.** E2e proof: trait with default method, impl that does not override it, call the default

## Feature 7: Stdlib Real Bodies

### Hasher
- [ ] **7a.** Implement `Hasher.write_u64`: FNV-1a style — `state = (state ^ value) * prime`
- [ ] **7b.** Verify `Hasher.finish`: return `self.state` (already done)

### List[T]
- [ ] **7c.** Implement `List.push`: check cap, call `fuse_rt_mem_realloc` if needed, write element via pointer arithmetic, increment len
- [ ] **7d.** Implement `List.get`: bounds check `index < len`, read element via pointer arithmetic, return `Some(elem)` or `None`
- [ ] **7e.** Implement `List.pop`: check `len > 0`, decrement len, read last element, return `Some(elem)` or `None`

### String
- [ ] **7f.** Implement `String.toUpper`: iterate bytes, if `97<=b<=122` subtract 32, build new String
- [ ] **7g.** Implement `String.toLower`: iterate bytes, if `65<=b<=90` add 32, build new String

### Formatter
- [ ] **7h.** Implement `Formatter.write_str`: copy string bytes into formatter buffer

### Map[K, V]
- [ ] **7i.** Implement `Map.insert/get/remove`: linear-probe hash table using Hasher + backing array

### Set[T]
- [ ] **7j.** Implement `Set.insert/contains/remove`: delegate to inner Map

### Verification
- [ ] **7k.** Compile all updated stdlib files, run full test suite
- [ ] **7l.** Commit and push
