package T {
    part def titled {
        rep language "adoc"
            /*
            = Element: sysml-name:[]

            Titled body.
            */
    }
    part def plain1 {
        rep language "adoc"
            /* No title here. */
    }
    part def plain2 {
        rep language "adoc"
            /* No title either. */
    }
    part def dupA {
        rep language "adoc"
            /*
            = Same

            First of two.
            */
    }
    part def dupB {
        rep language "adoc"
            /*
            = Same

            Second of two.
            */
    }
    part def two {
        rep language "adoc"
            /* Untitled first representation. */
        rep language "adoc"
            /*
            = Second

            Second body.
            */
        rep language "adoc"
            /*
            = Third

            Third body.
            */
    }
    part u : titled;
    part def markup {
        rep language "adoc"
            /*
            = A *bold* <thing>

            Markup body.
            */
    }
}
