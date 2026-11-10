// Failing macros (MC-19): each leaves a placeholder on every page that
// shows it and a warning with the file and line where it is written, and
// the run goes on, so every failing page is reported.
//
// - sysml-value:mas[] names an attribute pd does not have. pd's
//   representation is shown on the pages of a and pd: a warning on each.
// - sysml-doc::Nothing[] names no element; it is in b's own representation.
//
// Expected, in page order (a is declared before pd):
//   macroerror.sysml:21: M::a:  sysml-value:mas[]
//   macroerror.sysml:21: M::pd: sysml-value:mas[]
//   macroerror.sysml:26: M::b:  sysml-doc::Nothing[]
package M {
    part a : pd;
    part def pd {
        attribute mass = 5;
        rep language "adoc"
            /*
            Mass is

            sysml-value:mas[].
            */
    }
    part b {
        rep language "adoc"
            /* sysml-doc::Nothing[] */
    }
}
