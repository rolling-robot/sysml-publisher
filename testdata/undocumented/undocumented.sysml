// ER-4: no element has an AsciiDoc representation, so the site is the
// landing page only. A doc comment is not a representation.
package Plain {
    part def pd {
        doc /* Documented by a comment, not by a representation. */
        attribute mass;
    }
    part p : pd;
}
