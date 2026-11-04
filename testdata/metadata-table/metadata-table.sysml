// Fixture for the block macro sysml-metadata-table::<metadata def>[], which shows
// the attributes of the named metadata on the page's element: each
// attribute's name and value, the metadata def's default where the element
// does not bind one. The // notes give the expected tables.
package Meta {
    private import ScalarValues::*;

    enum def Level { low; high; }

    // Every value shape: string, integer, real, boolean, enumeration
    // literal, expression, a fixed value (= instead of default), and an
    // attribute with no value at all.
    metadata def Info {
        attribute owner : String default "nobody";
        attribute count : Integer default 3;
        attribute ratio : Real = 0.5;
        attribute ok : Boolean default true;
        attribute level : Level default Level::low;
        attribute sum : Integer default 1 + 2;
        attribute note : String;
    }

    // A metadata def that specializes another: @Derived shows the
    // attributes of both.
    metadata def Base {
        attribute origin : String default "base";
    }
    metadata def Derived :> Base {
        attribute extra : String default "x";
    }

    // Every page in this package shows the Info table and the Derived table
    // of its own element.
    part def Documented {
        rep language "asciidoc"
            /*
            = sysml-name:[]

            sysml-metadata-table::Info[]

            sysml-metadata-table::Derived[]
            */
    }

    // Some attributes bound, the rest from the defaults.
    // Info: owner = pete, count = 7, ratio = 0.5, ok = true,
    //       level = Level::low, sum = 1 + 2, note = (no value)
    // Derived: none, so no table.
    part bound : Documented {
        @Info {
            owner = "pete";
            count = 7;
        }
    }

    // Every attribute bound, overriding every default.
    // Info: owner = ann, count = 0, ratio = 0.5 (fixed in the def),
    //       ok = false, level = Level::high, sum = 10, note = checked
    // Derived: none.
    part overridden : Documented {
        @Info {
            owner = "ann";
            count = 0;
            ok = false;
            level = Level::high;
            sum = 10;
            note = "checked";
        }
    }

    // Prefix form, nothing bound: all defaults.
    // Info: owner = nobody, count = 3, ratio = 0.5, ok = true,
    //       level = Level::low, sum = 1 + 2, note = (no value)
    // Derived: none.
    #Info part defaults : Documented;

    // Metadata whose def specializes another, one inherited attribute bound.
    // Info: none, so no table.
    // Derived: extra = x, origin = here
    part specialized : Documented {
        @Derived {
            origin = "here";
        }
    }

    // No metadata: both macros give nothing.
    part plain : Documented;
}
