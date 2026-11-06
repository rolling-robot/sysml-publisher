// sysml-rep:[<element>][<language>] (MC-16): the body of an element's own
// textual representation in a language, as plain text.
//
// On R::Engine's page:
//   sysml-rep:[python]         power = torque * speed
//   sysml-rep:["python"]       the same (the language may be quoted)
//   sysml-rep:[LATEX]          both latex representations, any letter case,
//                              joined in declaration order
//   sysml-rep:Other[ocl]       another element, which has no page; escaped:
//                              self.mass &gt; 0 and a &lt; b
//   sysml-rep:R::Other[OCL]    the same, by qualified name
// On R::e's page (e : Engine):
//   sysml-rep:E[python]        Engine by its short name: power = torque * speed
//   sysml-rep:[python]         error: e has no python representation of its
//                              own (inherited ones do not count)
// Errors: sysml-rep:Other[] (no language), sysml-rep:Nothing[ocl].
package R {
    part def <E> Engine {
        rep language "adoc"
            /* The engine. */
        rep language "python"
            /* power = torque * speed */
        rep language "latex"
            /* P = \tau \omega */
        rep language "LaTeX"
            /* \omega = 2 \pi n */
    }
    part e : Engine;
    part def Other {
        rep language "ocl"
            /* self.mass > 0 and a < b */
    }
}
