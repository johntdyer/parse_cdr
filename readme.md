### Tropo CDR Parsing Tool

Tool to process Aspect's usage reports in XML format

#### Usage

    ./parse_cdr -infile="example.xml"
    +---------------------+-----------+-----------+-----------+
    |      CATEGORY       |  INBOUND  | OUTBOUND  |   TOTAL   |
    +---------------------+-----------+-----------+-----------+
    | CDR Counts          | 1,234,567 | 1,234,567 | 2,469,134 |
    | Duration (Minutes)  | 1,234,567 | 1,234,567 | 2,469,134 |
    +---------------------+-----------+-----------+-----------+
    +---------------+-----------+-----------+------------+
    |   CATEGORY    |  INBOUND  | OUTBOUND  |   TOTAL    |
    +---------------+-----------+-----------+------------+
    | Transport     |  12345.67 |  12345.67 |   24691.34 |
    | Platform      |   1234.56 |   1234.56 |    2469.12 |
    | Payphone      |      0.00 |      0.00 |       0.00 |
    | Transfer      |      0.00 |      0.00 |       0.00 |
    | Recording     |      0.00 |      0.00 |       0.00 |
    | Conferencing  |      0.00 |      0.00 |       0.00 |
    +---------------+-----------+-----------+------------+
    | TOTAL CHARGES | $13580 23 | $13580 23 |  $27160 46 |
    +---------------+-----------+-----------+------------+


####  Todo:

* Summerize outbound dialing destinations
