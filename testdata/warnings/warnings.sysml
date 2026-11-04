// Macro warnings: one per page and macro (MC-7).
//
// position and home are composite, so each sysml-value of them warns.
// - Pump's representation names position twice: one warning on each page.
// - Tagged :> Pump has its own representation naming position too: Tagged's
//   page and a's page show both representations, still one position warning.
// - home is a different macro: a warning of its own on the same pages.
// - Pump's representation is shown on four pages: each page warns.
//
// Expected, in page order:
//   W::Pump:   position, home
//   W::Tagged: position, home
//   W::a:      position, home
//   W::b:      position, home
package W {
    attribute def Pos {
        attribute x;
        attribute y;
    }
    part def Pump {
        attribute position : Pos;
        attribute home : Pos;
        rep language "adoc"
            /*
            At sysml-value:position[], still sysml-value:position[].

            Home: sysml-value:home[].
            */
    }
    part def Tagged :> Pump {
        rep language "adoc"
            /* Tagged at sysml-value:position[]. */
    }
    part a : Tagged;
    part b : Pump;
}
