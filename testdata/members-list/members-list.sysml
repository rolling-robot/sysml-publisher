// sysml-members-list (MC-12). Expected lists, as "name (types)" items:
//
// On M::pd's page:
//   sysml-members-list::[]   every owned member, in declaration order, with
//                            its types; no docs, representations, metadata:
//     mass (Real), a*b*+++c, sub (linked), Unnamed (other), wheels (Wheel),
//     front, spare (Sport), Racing, both (Wheel, Sport), link
//   ::Wheel[]   members typed by, subclassing or subsetting Wheel, at any
//               depth, without types: wheels (: Wheel), front (:> wheels,
//               which is : Wheel), spare (: Sport, Sport :> Wheel), Racing
//               (Racing :> Sport :> Wheel), both (: Wheel, Sport)
//   ::M::Sport[]       by qualified name: spare, Racing, both
//   ::other[]          typed only: Unnamed (part : other)
//   ::wheels[]         relative to pd (M::pd::wheels): front
//   ::sub[]            nothing specializes sub: the line is dropped
//   ::ScalarValues::Real[]  a library element: mass (: Real)
//   Short names, in every segment and every lookup:
//   ::W[]              Wheel's short name: as ::Wheel[]
//   ::MM::W[]          both segments short: as ::M::Wheel[]
//   ::ws[]             relative, short: M::pd::wheels: front
//   ::pd::ws[]         relative from M, short last segment: front
//   Element kinds, when no element has the name (any letter case, exact
//   metaclass: an interface is no PartUsage here):
//   ::partUsage[]      sub, Unnamed, wheels, front, spare, both
//   ::InterfaceUsage[] link
//   ::attributeusage[] mass
//   ::ActionUsage[]    a kind pd has no member of: the line is dropped
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
// Errors: ::Nothing[] (no element or kind), ::x[] (two elements named x;
// not read as a kind).
package <MM> M {
    private import ScalarValues::*;
    metadata def tag;
    part def other;
    part def <W> Wheel;
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
        part <ws> wheels : Wheel[4];
        part front :> wheels;
        part spare : Sport;
        part def Racing :> Sport;
        part both : Wheel, Sport;
        interface link connect 'a*b*+++c' to 'a*b*+++c';
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
