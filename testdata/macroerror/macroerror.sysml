// A representation with a broken macro: sysml-value names an attribute that
// does not exist. Rendering fails on the first page that shows it.
//
// a is declared before pd, so a's page is converted first: the error names
// the owner of the representation (M::pd) and the page (M::a), which differ.
package M {
    part a : pd;
    part def pd {
        attribute mass = 5;
        rep language "adoc"
            /* Mass: sysml-value:mas[]. */
    }
}
