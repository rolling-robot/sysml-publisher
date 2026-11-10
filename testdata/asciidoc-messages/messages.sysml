// Asciidoctor's own messages (CLI-6): each is a log record with the page's
// element and the model line it is about, not a raw line on stderr.
//
// pd's representation skips section levels, so Asciidoctor warns "section
// title out of sequence" about its line 3 (the ==== line, line 18 of this
// file). It is shown on the pages of pd and of u : pd: one record each.
//
// Expected, in page order (pd is declared before u):
//   WARN asciidoc: section title out of sequence: expected level 1, got level 3
//        element=M::pd location=<this file>:18
//   the same with element=M::u
package M {
    part def pd {
        rep language "adoc"
            /*
            Intro.

            ==== Skipped levels
            */
    }
    part u : pd;
}
