/**
 * Generates HomeLog ER diagram in draw.io XML format
 * using Atzeni notation (Basi di Dati - Atzeni, Ceri, Paraboschi, Torlone)
 *
 * Notation:
 * - Entity: rectangle
 * - Relationship: diamond (rhombus)
 * - Attribute: small ellipse connected by line, name as label
 * - Identifier: underlined text + filled dot on the line (entity side)
 * - Cardinality: (min,max) label on entity-relationship lines
 * - Generalization: filled triangle arrow pointing to parent
 *
 * Layout rules:
 * - 3 horizontal bands: Users/Properties, Expenses, Utilities
 * - Only orthogonal (90°) lines — no diagonals
 * - Attributes placed on the "quiet" side of entities (away from relationships)
 * - Generous spacing to avoid overlaps
 */

import { writeFileSync } from 'fs';

// ── Helpers ──────────────────────────────────────────────

let nextId = 100;
const id = () => `c${nextId++}`;

const esc = s =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

// ── Styles ───────────────────────────────────────────────

const S = {
  entity:
    'rounded=0;whiteSpace=wrap;html=1;fillColor=#dae8fc;strokeColor=#6c8ebf;strokeWidth=2;fontSize=14;fontStyle=1;',
  rel:
    'shape=rhombus;perimeter=rhombusPerimeter;whiteSpace=wrap;html=1;fillColor=#fff2cc;strokeColor=#d6b656;strokeWidth=1.5;fontSize=11;fontStyle=0;',
  attr:
    'ellipse;whiteSpace=wrap;html=1;fillColor=#f5f5f5;strokeColor=#999999;strokeWidth=1;fontSize=9;',
  attrKey:
    'ellipse;whiteSpace=wrap;html=1;fillColor=#f5f5f5;strokeColor=#999999;strokeWidth=1;fontSize=9;fontStyle=4;',
  // Orthogonal lines — the key change: edgeStyle=orthogonalEdgeStyle + rounded=1
  line:
    'edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;endArrow=none;endFill=0;strokeWidth=1.2;strokeColor=#999999;',
  lineIdent:
    'edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;endArrow=none;endFill=0;strokeWidth=1.2;strokeColor=#999999;startFill=1;startSize=6;startArrow=oval;',
  relLine:
    'edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;endArrow=none;endFill=0;strokeWidth=1.5;strokeColor=#666666;',
  cardLabel:
    'edgeLabel;html=1;align=center;verticalAlign=middle;resizable=0;points=[];fontSize=10;fontStyle=1;fontColor=#c0392b;',
  genArrow:
    'edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;endArrow=block;endFill=1;endSize=14;strokeWidth=2;strokeColor=#666666;',
  genChild:
    'rounded=0;whiteSpace=wrap;html=1;fillColor=#d5e8d4;strokeColor=#82b366;strokeWidth=2;fontSize=13;fontStyle=1;',
  sectionLabel:
    'text;html=1;strokeColor=none;fillColor=none;align=left;verticalAlign=middle;whiteSpace=wrap;rounded=0;fontSize=18;fontColor=#aaaaaa;fontStyle=0;',
};

// ── Cell builders ────────────────────────────────────────

function vertex(cellId, value, style, x, y, w, h) {
  return `    <mxCell id="${cellId}" value="${esc(value)}" style="${style}" vertex="1" parent="1">
      <mxGeometry x="${x}" y="${y}" width="${w}" height="${h}" as="geometry"/>
    </mxCell>`;
}

function edgeCell(cellId, src, tgt, style, points) {
  const pts = points
    ? `<Array as="points">${points.map(([px, py]) => `<mxPoint x="${px}" y="${py}"/>`).join('')}</Array>`
    : '';
  return `    <mxCell id="${cellId}" value="" style="${style}" edge="1" source="${src}" target="${tgt}" parent="1">
      <mxGeometry relative="1" as="geometry">${pts}</mxGeometry>
    </mxCell>`;
}

function labelCell(parentEdgeId, text, pos, offsetX, offsetY) {
  const lblId = id();
  return `    <mxCell id="${lblId}" value="${esc(text)}" style="${S.cardLabel}" vertex="1" connectable="0" parent="${parentEdgeId}">
      <mxGeometry x="${pos}" y="0" relative="1" as="geometry"><mxPoint x="${offsetX || 0}" y="${offsetY || -12}" as="offset"/></mxGeometry>
    </mxCell>`;
}

// ── Layout constants ─────────────────────────────────────

// Entity size
const EW = 160; // entity width
const EH = 50;  // entity height

// Diamond size
const DW = 110;
const DH = 60;

// Attribute size
const AW = 65;
const AH = 26;

// Spacing
const COL_GAP = 300;  // horizontal gap between entity centers
const ROW_GAP = 280;  // vertical gap between entity centers
const ATTR_GAP = 42;  // gap between attributes
const ATTR_DIST = 70; // distance from entity to attribute center

// ── Entity positions ─────────────────────────────────────
// Organized in 3 bands, each with generous spacing

// Band 1: Users & Properties (y: 100–550)
const BAND1_Y = 100;
const layout = {
  // Row 1
  eUser:       [200,  BAND1_Y],
  eProperty:   [700,  BAND1_Y],

  // Row 2
  eSettings:   [200,  BAND1_Y + ROW_GAP],
  eHMember:    [700,  BAND1_Y + ROW_GAP],
  eHSettings:  [1150, BAND1_Y + ROW_GAP],

  // Band 2: Expenses (y: 700–1100)
  eCategory:    [200,  750],
  eSubcategory: [200,  1030],
  eExpense:     [700,  750],
  eSplit:       [1150, 750],
  eProject:     [700,  1030],
  eSettlement:  [1150, 1030],

  // Band 3: Utilities (y: 1350–2200)
  eUtility:     [200,  1400],
  eReading:     [650,  1400],
  eBill:        [650,  1680],
  eRate:        [1100, 1400],
  ePriceChg:    [1100, 1680],
  eComm:        [1100, 1940],
  eBillTpl:     [200,  1940],
  eContractTpl: [200,  2180],

  // Generalization children
  eMetered:     [100,  1640],
  eFixed:       [330,  1640],
};

// ── Attributes ───────────────────────────────────────────
// [entityId, name, isKey, side, index]
// side: 'top' | 'bottom' | 'left' | 'right' — where to place relative to entity
// index: 0-based position along that side

const attrs = [
  // USER — attrs on top (relationships go right, down, left)
  ['eUser', 'email', true, 'top', 0],
  ['eUser', 'name', false, 'top', 1],
  ['eUser', 'role', false, 'top', 2],
  ['eUser', 'is_active', false, 'top', 3],

  // USER_SETTINGS — attrs on bottom
  ['eSettings', 'language', false, 'bottom', 0],
  ['eSettings', 'currency', false, 'bottom', 1],
  ['eSettings', 'theme', false, 'bottom', 2],

  // PROPERTY — attrs on top
  ['eProperty', 'name', true, 'top', 0],
  ['eProperty', 'address', false, 'top', 1],
  ['eProperty', 'type', false, 'top', 2],
  ['eProperty', 'is_current', false, 'top', 3],

  // HOUSEHOLD_MEMBER — attrs on right
  ['eHMember', 'name', true, 'right', 0],
  ['eHMember', 'role', false, 'right', 1],
  ['eHMember', 'is_virtual', false, 'right', 2],

  // HOUSEHOLD_SETTINGS — attrs on right
  ['eHSettings', 'split_mode', false, 'right', 0],
  ['eHSettings', 'default_split_type', false, 'right', 1],

  // CATEGORY — attrs on left
  ['eCategory', 'name', true, 'left', 0],
  ['eCategory', 'icon', false, 'left', 1],
  ['eCategory', 'color', false, 'left', 2],

  // SUBCATEGORY — attrs on left
  ['eSubcategory', 'name', true, 'left', 0],

  // EXPENSE — attrs on top
  ['eExpense', 'amount', false, 'top', 0],
  ['eExpense', 'date', false, 'top', 1],
  ['eExpense', 'description', false, 'top', 2],
  ['eExpense', 'is_split', false, 'top', 3],

  // EXPENSE_SPLIT — attrs on top
  ['eSplit', 'amount', false, 'top', 0],
  ['eSplit', 'is_settled', false, 'top', 1],

  // SETTLEMENT — attrs on right
  ['eSettlement', 'amount', false, 'right', 0],
  ['eSettlement', 'date', false, 'right', 1],
  ['eSettlement', 'payment_method', false, 'right', 2],

  // PROJECT — attrs on bottom
  ['eProject', 'name', true, 'bottom', 0],
  ['eProject', 'budget', false, 'bottom', 1],
  ['eProject', 'status', false, 'bottom', 2],

  // UTILITY — attrs on left
  ['eUtility', 'type', false, 'left', 0],
  ['eUtility', 'provider', false, 'left', 1],
  ['eUtility', 'service_code', true, 'left', 2],
  ['eUtility', 'is_active', false, 'left', 3],

  // METER_READING — attrs on top
  ['eReading', 'reading_date', true, 'top', 0],
  ['eReading', 'value', false, 'top', 1],
  ['eReading', 'source', false, 'top', 2],

  // BILL — attrs on bottom
  ['eBill', 'bill_number', true, 'bottom', 0],
  ['eBill', 'amount_total', false, 'bottom', 1],
  ['eBill', 'due_date', false, 'bottom', 2],
  ['eBill', 'is_paid', false, 'bottom', 3],

  // UTILITY_RATE — attrs on top
  ['eRate', 'effective_date', true, 'top', 0],
  ['eRate', 'rate_unit', false, 'top', 1],

  // PRICE_CHANGE — attrs on right
  ['ePriceChg', 'effective_date', true, 'right', 0],
  ['ePriceChg', 'old_amount', false, 'right', 1],
  ['ePriceChg', 'new_amount', false, 'right', 2],

  // SERVICE_COMMUNICATION — attrs on right
  ['eComm', 'type', false, 'right', 0],
  ['eComm', 'title', true, 'right', 1],
  ['eComm', 'is_important', false, 'right', 2],

  // BILL_TEMPLATE — attrs on bottom
  ['eBillTpl', 'name', true, 'bottom', 0],
  ['eBillTpl', 'provider', false, 'bottom', 1],
  ['eBillTpl', 'extraction_rules', false, 'bottom', 2],

  // CONTRACT_TEMPLATE — attrs on bottom
  ['eContractTpl', 'name', true, 'bottom', 0],
  ['eContractTpl', 'provider', false, 'bottom', 1],
];

// ── Relationship definitions ─────────────────────────────
// [relId, relName, ent1, card1, ent2, card2]
// Diamond is auto-positioned midway between ent1 and ent2

const rels = [
  // Band 1: Users & Properties
  ['rHa',        'ha',          'eUser',     '(1,1)', 'eSettings',  '(1,1)'],
  ['rPossiede',  'possiede',    'eUser',     '(0,N)', 'eProperty',  '(1,1)'],
  ['rInclude',   'include',     'eProperty', '(0,N)', 'eHMember',   '(1,1)'],
  ['rE',         'è',           'eUser',     '(0,N)', 'eHMember',   '(0,1)'],
  ['rConfigura', 'configura',   'eProperty', '(0,1)', 'eHSettings', '(1,1)'],

  // Band 2: Expenses
  ['rHaCat',      'ha',           'eCategory',   '(0,N)', 'eSubcategory', '(1,1)'],
  ['rRegistra',   'registra',     'eUser',       '(0,N)', 'eExpense',     '(1,1)'],
  ['rClassifica', 'classifica',   'eCategory',   '(0,N)', 'eExpense',     '(1,1)'],
  ['rPagata',     'pagata da',    'eHMember',    '(0,N)', 'eExpense',     '(1,1)'],
  ['rSuddivisa',  'suddivisa in','eExpense',    '(0,N)', 'eSplit',       '(1,1)'],
  ['rQuotaDi',    'quota di',    'eHMember',    '(0,N)', 'eSplit',       '(1,1)'],
  ['rSalda',      'salda',       'eSettlement', '(0,N)', 'eSplit',       '(0,1)'],
  ['rDa',         'da',          'eHMember',    '(0,N)', 'eSettlement',  '(1,1)'],
  ['rA',          'a',           'eHMember',    '(0,N)', 'eSettlement',  '(1,1)'],
  ['rCrea',       'crea',        'eUser',       '(0,N)', 'eProject',     '(1,1)'],
  ['rInProg',     'in progetto', 'eProject',    '(0,N)', 'eExpense',     '(0,1)'],
  ['rCondiviso',  'condiviso',   'eProject',    '(0,N)', 'eUser',        '(0,N)'],

  // Band 3: Utilities
  ['rGestisce',   'gestisce',     'eUser',      '(0,N)', 'eUtility',    '(1,1)'],
  ['rServita',    'servita da',   'eProperty',  '(0,N)', 'eUtility',    '(1,1)'],
  ['rRileva',     'rileva',       'eUtility',   '(0,N)', 'eReading',    '(1,1)'],
  ['rFattura',    'fattura',      'eUtility',   '(0,N)', 'eBill',       '(1,1)'],
  ['rTariffa',    'tariffa',      'eUtility',   '(0,N)', 'eRate',       '(1,1)'],
  ['rVariazione', 'variazione',   'eUtility',   '(0,N)', 'ePriceChg',   '(1,1)'],
  ['rComunica',   'comunica',     'eUtility',   '(0,N)', 'eComm',       '(1,1)'],
  ['rUsaTpl',     'usa',          'eUtility',   '(0,1)', 'eBillTpl',    '(0,N)'],
  ['rAssociata',  'associata a',  'eBill',      '(0,1)', 'eReading',    '(0,1)'],
  ['rDaBollPc',   'da bolletta',  'ePriceChg',  '(0,1)', 'eBill',       '(0,N)'],
  ['rDaBollCm',   'da bolletta',  'eComm',      '(0,1)', 'eBill',       '(0,N)'],
  ['rCreaTpl',    'crea',         'eUser',      '(0,N)', 'eBillTpl',    '(1,1)'],
  ['rCreaCTpl',   'crea',         'eUser',      '(0,N)', 'eContractTpl','(1,1)'],
];

// ── Build XML cells ──────────────────────────────────────

const cells = [];

// Section labels
cells.push(vertex('sec1', 'UTENTI E PROPRIETÀ', S.sectionLabel, 100, 40, 300, 30));
cells.push(vertex('sec2', 'SPESE E BILANCIO', S.sectionLabel, 100, 690, 300, 30));
cells.push(vertex('sec3', 'SERVIZI (UTENZE)', S.sectionLabel, 100, 1340, 300, 30));

// Entities
for (const [eid, pos] of Object.entries(layout)) {
  const entDef = {
    eUser: 'USER', eSettings: 'USER_SETTINGS', eProperty: 'PROPERTY',
    eHMember: 'HOUSEHOLD_MEMBER', eHSettings: 'HOUSEHOLD_SETTINGS',
    eCategory: 'CATEGORY', eSubcategory: 'SUBCATEGORY',
    eExpense: 'EXPENSE', eSplit: 'EXPENSE_SPLIT',
    eSettlement: 'SETTLEMENT', eProject: 'PROJECT',
    eUtility: 'UTILITY', eReading: 'METER_READING',
    eBill: 'BILL', eRate: 'UTILITY_RATE',
    ePriceChg: 'PRICE_CHANGE', eComm: 'SERVICE_COMMUNICATION',
    eBillTpl: 'BILL_TEMPLATE', eContractTpl: 'CONTRACT_TEMPLATE',
    eMetered: 'SERVIZIO A CONTATORE', eFixed: 'SERVIZIO A CANONE',
  };
  const isGen = eid === 'eMetered' || eid === 'eFixed';
  const w = entDef[eid].length > 16 ? 200 : EW;
  cells.push(vertex(eid, entDef[eid], isGen ? S.genChild : S.entity, pos[0], pos[1], w, EH));
}

// Attributes
for (const [entId, name, isKey, side, idx] of attrs) {
  const [ex, ey] = layout[entId];
  const entName = entId; // for width calc
  const ew = name.length > 16 ? 200 : EW;

  let ax, ay;
  const centerX = ex + ew / 2;
  const centerY = ey + EH / 2;

  // Calculate total attrs on this side for centering
  const samesSide = attrs.filter(a => a[0] === entId && a[3] === side);
  const total = samesSide.length;
  const offset = (idx - (total - 1) / 2) * ATTR_GAP;

  switch (side) {
    case 'top':
      ax = centerX + offset - AW / 2;
      ay = ey - ATTR_DIST - AH / 2;
      break;
    case 'bottom':
      ax = centerX + offset - AW / 2;
      ay = ey + EH + ATTR_DIST - AH / 2;
      break;
    case 'left':
      ax = ex - ATTR_DIST - AW;
      ay = centerY + offset - AH / 2;
      break;
    case 'right':
      ax = ex + ew + ATTR_DIST;
      ay = centerY + offset - AH / 2;
      break;
  }

  const aId = id();
  const lineId = id();
  cells.push(vertex(aId, name, isKey ? S.attrKey : S.attr, ax, ay, AW, AH));
  cells.push(edgeCell(lineId, entId, aId, isKey ? S.lineIdent : S.line));
}

// Relationships
for (const [rId, name, e1, c1, e2, c2] of rels) {
  const [x1, y1] = layout[e1];
  const [x2, y2] = layout[e2];

  const ew1 = EW;
  const ew2 = EW;

  // Place diamond midway
  const mx = (x1 + ew1 / 2 + x2 + ew2 / 2) / 2 - DW / 2;
  const my = (y1 + EH / 2 + y2 + EH / 2) / 2 - DH / 2;

  cells.push(vertex(rId, name, S.rel, mx, my, DW, DH));

  // Line: entity1 → diamond
  const l1 = id();
  cells.push(edgeCell(l1, e1, rId, S.relLine));
  cells.push(labelCell(l1, c1, 0.15, 0, -12));

  // Line: diamond → entity2
  const l2 = id();
  cells.push(edgeCell(l2, rId, e2, S.relLine));
  cells.push(labelCell(l2, c2, 0.85, 0, -12));
}

// Generalization arrows
const g1 = id();
const g2 = id();
cells.push(edgeCell(g1, 'eMetered', 'eUtility', S.genArrow));
cells.push(edgeCell(g2, 'eFixed', 'eUtility', S.genArrow));

// ── Assemble XML ─────────────────────────────────────────

const xml = `<?xml version="1.0" encoding="UTF-8"?>
<mxfile host="app.diagrams.net" type="device" compressed="false">
  <diagram id="homelog-er" name="HomeLog ER (Atzeni)">
    <mxGraphModel dx="3000" dy="2600" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="0" pageScale="1" pageWidth="5000" pageHeight="4000" math="0" shadow="0">
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>
${cells.join('\n')}
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
`;

writeFileSync('docs/er-diagram.drawio', xml, 'utf8');
console.log('Generated docs/er-diagram.drawio');
console.log(`  ${Object.keys(layout).length} entities`);
console.log(`  ${attrs.length} attributes`);
console.log(`  ${rels.length} relationships`);
