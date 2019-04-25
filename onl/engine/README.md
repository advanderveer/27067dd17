# TODO

Fixes:
1. Commit the chains tip as part of the store tx, else the tip might race with the
   actual block being there.
2. Test result not deterministic in rare cases
3. OutOfOrder needs to be thread safe
4. Should reach consensus on three writer setup

Enhancements:
1. Add a .LastError() method that can be asserted to see everything went well
