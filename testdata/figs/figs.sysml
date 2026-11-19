// Fixture for the block macro sysml-fig::[<element>][], replaced by a Mermaid
// figure of an interface: the parts on its ends, up to their common ancestor,
// the end features, and one edge per pair of ends. The // notes state what
// each page shows. The figure's look is OpenSysML's Mermaid interconnection
// form; the tests read the generated Mermaid text, not a picture.
package F {
    port def P;

    part vehicle {
        rep language "asciidoc"
            /* = Vehicle

               sysml-fig::i1[]

               sysml-fig::i2[]

               sysml-fig::ixy[]

               sysml-fig::Itself[]

               sysml-fig::nope[] */
        // a owns p and, nested in it, sub with deep; inner with r and loose
        // are on no end of i1 or i2, so no figure of them shows them.
        part a {
            port p : P;
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
        // i1 connects a and b: the figure shows a (with p) and b (with q),
        // the edge p === q. c, inner, loose and i2 are not in it.
        interface i1 connect (a.p, b.q) {
            rep language "asciidoc"
                /* = i1 */
        }
        // i2 connects a.sub.deep and b.q: the figure shows a > sub (with
        // deep) and b (with q), the edge deep === q. inner and loose are not
        // in it.
        interface i2 connect (a.sub.deep, b.q) {
            rep language "asciidoc"
                /* = i2 */
        }
        // c is on no interface: as a target it is written out, no warning.
    }

    // An interface between parts of different packages: their chains share
    // only F, a package, which is not drawn; the figure shows px (with p)
    // and py (with p) as loose boxes, the edge p === p.
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
