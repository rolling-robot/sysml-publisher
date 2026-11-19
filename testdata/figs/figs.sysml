// Fixture for the block macro sysml-fig::[<element>][], replaced by a Mermaid
// figure of an interface or of a part: an interface shows the parts on its
// ends, up to their common ancestor, the end features, and one edge per pair
// of ends; a part shows its owned ports, and, for every connection they are
// on, the other ends' features and their owner parts. The // notes state
// what each page shows. The figure's look is OpenSysML's Mermaid
// interconnection form; the tests read the generated Mermaid text, not a
// picture.
package F {
    port def P;

    part vehicle {
        rep language "asciidoc"
            /* = Vehicle

               sysml-fig::i1[]

               sysml-fig::i2[]

               sysml-fig::ixy[]

               sysml-fig::Itself[]

               sysml-fig::nope[]

               sysml-fig::a[]

               sysml-fig::c[]

               sysml-fig::d[]

               sysml-fig::e[]

               sysml-fig::vehicle[] */
        // a owns p and, nested in it, sub with deep; inner with r and loose
        // are on no end of i1 or i2, so no figure of them shows them. a and
        // its port p carry pages of their own, so the figures of i1, i2 and
        // a link a's box and p's node to them.
        part a {
            rep language "asciidoc"
                /* = A */
            port p : P {
                rep language "asciidoc"
                    /* = p */
            }
            part sub {
                port deep : P;
                part loose {
                }
            }
            part inner {
                port r : P;
            }
        }
        part b {
            port q : P;
        }
        part c {
            port s : P;
        }
        // c is on no interface: as a target the figure of c shows c and its
        // port s, no edges.
        // i1 connects a and b: the figure shows vehicle holding a (with p)
        // and b (with q), the edge p === q. c, inner, loose and i2 are not
        // in it.
        interface i1 connect (a.p, b.q) {
            rep language "asciidoc"
                /* = i1 */
        }
        // i2 connects a.sub.deep and b.q: the figure shows vehicle > a >
        // sub (with deep) and vehicle > b (with q), the edge deep === q.
        // inner and loose are not in it.
        interface i2 connect (a.sub.deep, b.q) {
            rep language "asciidoc"
                /* = i2 */
        }
        // d owns pd, and da with pa; idd connects pd to da.pa: the figure of
        // d shows d with pd, and da (with pa) inside it — da is d's own —
        // the edge pd === pa.
        part d {
            port pd : P;
            part da {
                port pa : P;
            }
            interface idd connect (pd, da.pa);
        }
        // ie connects e's own two ports: the figure of e shows e with e1
        // and e2, the edge e1 === e2.
        part e {
            port e1 : P;
            port e2 : P;
            interface ie connect (e1, e2);
        }
        // vehicle has no ports: as a target the macro fails.
    }

    // An interface between parts of different packages: their chains share
    // only F, a package, which is the common ancestor and drawn as the
    // figure's frame; the figure shows F holding px (with p) and py (with
    // p), the edge p === p.
    package X {
        part px {
            port p : P;
        }
    }
    package Y {
        part py {
            port p : P;
        }
    }
    interface ixy connect (X::px.p, Y::py.p);

    // An interface def is not an interface usage: as a target it is written
    // out, no warning.
    interface def Itself {
        end a : P;
        end b : P;
    }

    // No such element: as a target the macro fails.
}
