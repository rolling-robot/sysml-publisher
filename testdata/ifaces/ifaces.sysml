// Fixture for the block macro sysml-ifaces::[<element>][], a bullet list of
// the interfaces and connections the target's owned ports are on: each item
// one connection by name, -> , and the distinct part(s) on the other ends.
// The // notes give the expected list on the page of the part named.
package I {
    port def P;

    // x owns p and q; x has a short name, ex, so its own page links to it by
    // ex. On x's page:
    //   i1 -> wy      (i1: x.p - y.p, y's short name)
    //   i2 -> z       (i2: x.q - z.p)
    //   i3 -> wy, z   (i3: x.p, y.p, z.p)
    //   c1 -> wy      (c1: a connection, not an interface)
    //   s1 ->         (self-loop x.p - x.q)
    // with name=long:
    //   i1 -> y
    //   i2 -> z
    //   i3 -> y, z
    //   c1 -> y
    //   s1 ->
    part <ex> x {
        rep language "asciidoc"
            /*
            sysml-ifaces::[]
            */
        port p : P;
        port q : P;
    }
    // y owns p; its short name is wy.
    part <wy> y {
        rep language "asciidoc"
            /* sysml-ifaces::[] */
        port p : P;
    }
    // z owns p; it has no short name, so it is shown by its name under the
    // default short too.
    part z {
        rep language "asciidoc"
            /* sysml-ifaces::[] */
        port p : P;
    }
    // n has no owned ports: the macro line is dropped (no <li>).
    part n {
        rep language "asciidoc"
            /* sysml-ifaces::[] */
    }
    // m owns a port nothing references: dropped.
    part m {
        rep language "asciidoc"
            /* sysml-ifaces::[] */
        port r : P;
    }

    // The connections; each has a page, so an item links to it.
    // On x's page: i1 -> wy
    interface i1 connect (x.p, y.p) {
        rep language "asciidoc"
            /* = sysml-name:[] */
    }
    // i2 and i3 also connect x's ports. i3 is n-ary.
    interface i2 connect x.q to z.p {
        rep language "asciidoc"
            /* */
    }
    interface i3 connect (x.p, y.p, z.p) {
        rep language "asciidoc"
            /* */
    }
    // A connection, not an interface: listed the same way.
    connection c1 connect x.p to y.p {
        rep language "asciidoc"
            /* */
    }
    // A self-loop on x: all ends reference x's ports, so the right side is
    // empty. On x's page: s1 ->
    interface s1 connect (x.p, x.q) {
        rep language "asciidoc"
            /* */
    }
    // x4 connects only y and z, not x: not listed on x's page.
    interface x4 connect (y.p, z.p) {
        rep language "asciidoc"
            /* */
    }

    // Not a part: a target naming it fails the macro.
    package pkg {}
}