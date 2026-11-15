// Fixture for the block macro sysml-iface-ends::[<element>][], a bullet
// list of an interface's ends: each end's connected feature's segments, the
// source end first. The // notes give the expected lists.
package I {
    port def P;

    // Its ends connect nothing: the list is their names.
    // supplier; consumer
    interface def D {
        rep language "asciidoc"
            /* sysml-iface-ends::[] */
        end port supplier : P;
        end port consumer : P;
    }
    interface def N3 { end port a : P; end port b : P; end port c : P; }
    part def Box { port p : P; port q : P; port r : P; port 'de-icing fluid' : P; }

    part sys {
        rep language "asciidoc"
            /* = sysml-name:[] */
        // x has a page and a short name, ex, so its segment is a link shown
        // as ex (short names are the default).
        part <ex> x : Box {
            rep language "asciidoc"
                /* = sysml-name:[] */
        }
        part y : Box;
        part z : Box;
        // source: x.p; target: y.p -> ex.p; y.p
        //   with name=long: x.p; y.p
        interface i1 connect (x.p, y.p);
        // source: x.q; target: y.q -> ex.q; y.q
        interface i2 connect x.q to y.q;
        // a: x.p; b: y.p; c: z.p -> ex.p; y.p; z.p
        interface i4 : N3 connect (a ::> x.p, b ::> y.p, c ::> z.p);
        // ex.p; y.p; z.p (n-ary, unnamed ends, no source)
        interface i5 connect (x.p, y.p, z.p);
        // Inherits D's representation, so it has a page.
        // supplier: x.p; consumer: y.p -> ex.p; y.p
        interface i6 : D connect x.p to y.p;
        // Ends written in a body, target before source: source first.
        // source: x.r; target: z.r -> ex.r; z.r
        interface i8 {
            end port target ::> z.r;
            end port source ::> x.r;
        }
        // source: x.de-icing fluid; target: y.de-icing fluid
        // -> ex.de-icing fluid; y.de-icing fluid
        interface i9 connect x.'de-icing fluid' to y.'de-icing fluid';
        // A connection, not an interface: listed the same way.
        // source: x.p; target: z.p -> x.p; z.p
        connection c1 connect x.p to z.p;
    }
}
