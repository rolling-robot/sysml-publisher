package UavKinds {
    private import ScalarValues::*;
    private import SI::*;
    private import UavDefinitions::*;

    part <U> uav : UavDefinitions::UavPart {
        doc /*
        A general 50 kg fixed-wing UAV: a small electric air vehicle with a
        payload bay, a command-and-control link, a data link, a CAN bus and a
        power line. The system model specialises this kind for the UAV it
        actually flies.
        */
        rep language "asciidoc"
        /*
        == Flight data

        [cols="1,1"]
        |===
        |cruise speed | sysml-value:cruiseSpeed[]
        |endurance | sysml-value:endurance[]
        |range | sysml-value:range[]
        |empty mass | sysml-value:emptyMass[]
        |payload capacity | sysml-value:payloadCapacity[]
        |===
        */
        attribute :>> mass = 50 [kg];
        attribute cruiseSpeed : ISQ::SpeedValue default 25 [m/s];
        attribute endurance : ISQ::DurationValue default 90 [min];
        attribute range : ISQ::LengthValue = endurance * cruiseSpeed;
        attribute emptyMass : ISQ::MassValue default 42 [kg];
        attribute payloadCapacity : ISQ::MassValue default 8 [kg];
    }
}
