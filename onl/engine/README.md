# TODO

1. Commit the chains tip as part of the store tx, else the tip might race with the
   actual block being there.
2. Test result not deterministic
3. OutOfOrder needs to be thread safe
