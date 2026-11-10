// A model read by another module through the public packages only
// (macros.Example): sysml.Open, macros.Register with the host's Pages.
// Car and Wheel have pages in the example; nothing else does.
package Vehicles {
    part def Wheel;
    part def Car {
        attribute mass = 1200;
        part front : Wheel;
        part rear : Wheel;
    }
}
