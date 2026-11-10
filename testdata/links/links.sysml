// Links between pages (MC-18), decided by the output (render.Options.Pages).
// On L::a's page, sysml-name:b[] links to L::b's page: ../b/index.html in
// the HTML site, whatever the given Pages says otherwise.
package L {
    part def a {
        rep language "adoc"
            /* See sysml-name:b[]. */
    }
    part def b {
        rep language "adoc"
            /* B. */
    }
}
