// sysml-members-list (MC-12). Expected lists, as "name (types)" items:
//
// On M::pd's page:
//   sysml-members-list::[]   every owned member, in declaration order, with
//                            its types; no docs, representations, metadata:
//     mass (Real), a*b*+++c, sub (linked), Unnamed (other), wheels (Wheel),
//     front, spare (Sport), Racing, both (Wheel, Sport)
//   ::Wheel[]   members that subclass or subset Wheel, at any depth, without
//               types: Racing (Racing :> Sport :> Wheel). Not wheels, spare
//               or both: being typed by Wheel is neither.
//   ::M::Sport[]       by qualified name: Racing
//   ::wheels[]         relative to pd (M::pd::wheels): front
//   ::sub[]            nothing specializes sub: the line is dropped
//   ::ScalarValues::Real[]  a library element: mass is typed by it, which
//                      is not subsetting: the line is dropped
// On M::leaf's page:
//   ::[]   only leaf's own member: q (pd's members are not leaf's)
// On M::pd::sub's page:
//   ::[]   sub owns only a representation: the line is dropped
// On M::Car's page:
//   ::front[]   relative to Car (M::Car::front, a redefinition of pd's
//               front): nothing specializes it
//   ::wheels[]  not a name inside Car: the plain name, unique in the model,
//               gives M::pd::wheels; Car's own member front redefines
//               pd::front, which subsets it: front
// Errors: ::Nothing[] (no element), ::x[] (two elements named x).
package M {
    private import ScalarValues::*;
    metadata def tag;
    part def other;
    part def Wheel;
    part def Sport :> Wheel;
    part def pd {
        doc /* pd documentation. */
        rep language "adoc"
            /* Members: */
        attribute mass : Real;
        port 'a*b*+++c';
        part sub {
            rep language "adoc"
                /* sub has a page. */
        }
        part : other;
        part wheels : Wheel[4];
        part front :> wheels;
        part spare : Sport;
        part def Racing :> Sport;
        part both : Wheel, Sport;
    }
    part leaf : pd {
        @tag;
        port q;
    }
    part def Car :> pd {
        part front :>> front;
    }
    part def A { part x; }
    part def B { part x; }
}
