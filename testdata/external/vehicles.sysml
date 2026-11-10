// A model read by another module through the public packages only
// (macros_test: Example, the custom macro tests): sysml.Open,
// macros.Register with the host's Pages, Registry.Inline and Block.
// Car and Wheel have pages in the tests; nothing else does.
package Vehicles {
    part def <W> Wheel;
    part def Car {
        attribute mass = 1200;
        part front : Wheel;
        part rear : Wheel;
    }
}
