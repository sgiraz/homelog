import { writeFileSync } from 'fs';

// ── Helpers ──────────────────────────────────────────────
let nextId = 100;
const id = () => `c${nextId++}`;
const esc = s => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

// ── Styles ───────────────────────────────────────────────
const S = {
  entity: 'rounded=0;whiteSpace=wrap;html=1;fillColor=#dae8fc;strokeColor=#6c8ebf;strokeWidth=2;fontSize=14;fontStyle=1;',
  rel: 'shape=rhombus;perimeter=rhombusPerimeter;whiteSpace=wrap;html=1;fillColor=#fff2cc;strokeColor=#d6b656;strokeWidth=1.5;fontSize=11;fontStyle=0;',
  attr: 'ellipse;whiteSpace=wrap;html=1;fillColor=#f5f5f5;strokeColor=#999999;strokeWidth=1;fontSize=9;',
  attrKey: 'ellipse;whiteSpace=wrap;html=1;fillColor=#f5f5f5;strokeColor=#999999;strokeWidth=1;fontSize=9;fontStyle=4;',
  line: 'edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;endArrow=none;strokeColor=#999999;strokeWidth=1.2;',
  lineIdent: 'edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;endArrow=none;startArrow=oval;startFill=1;startSize=6;strokeColor=#999999;strokeWidth=1.2;',
  relLine: 'edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=40;jumpStyle=arc;html=1;endArrow=none;strokeWidth=1.5;strokeColor=#666666;',
  cardLabel: 'edgeLabel;html=1;align=center;verticalAlign=middle;resizable=0;points=[];fontSize=10;fontStyle=1;fontColor=#c0392b;labelBackgroundColor=#ffffff;',
  genArrow: 'edgeStyle=orthogonalEdgeStyle;rounded=1;orthogonalLoop=1;jettySize=auto;html=1;endArrow=block;endFill=1;endSize=14;strokeWidth=2;strokeColor=#666666;',
  genChild: 'rounded=0;whiteSpace=wrap;html=1;fillColor=#d5e8d4;strokeColor=#82b366;strokeWidth=2;fontSize=13;fontStyle=1;',
  sectionLabel: 'text;html=1;strokeColor=none;fillColor=none;align=left;verticalAlign=middle;whiteSpace=wrap;rounded=0;fontSize=18;fontColor=#aaaaaa;fontStyle=0;',
};

// ── Configurazione Dimensioni ────────────────────────────
const EW = 160; const EH = 50;
const DW = 110; const DH = 60;
const AW = 65;  const AH = 26;

// ── Layout a Corridoi Sfalsati (Anti-Sovrapposizione) ────
// Nessun elemento è allineato sullo stesso asse a meno che non sia direttamente adiacente.
// X=800 è un "canale vuoto" che permette a Property di collegarsi a Utility senza colpire Expense.
const layout = {
  eUser:        [400,  200],
  eProperty:    [1200, 200],  // Spostato a destra per non coprire Utility
  
  eSettings:    [200,  600],
  eHMember:     [1600, 600],  // Spostato per liberare l'asse X=1200
  eHSettings:   [2400, 600],

  eCategory:    [200,  1000],
  eExpense:     [1600, 1000], // Allineato a HMember, sicuro.
  eSplit:       [2400, 1000],
  
  eSubcategory: [200,  1400],
  eProject:     [1200, 1400],
  eSettlement:  [2400, 1400],

  eUtility:     [800,  2200], // Asse esclusivo! La linea scende senza toccare nulla
  eReading:     [1600, 1900], // Sfalsato in Y per i collegamenti laterali
  eRate:        [2400, 2200],
  
  eBill:        [1600, 2600],
  ePriceChg:    [2400, 2600],
  
  eBillTpl:     [400,  2600],
  eComm:        [2400, 3000],
  eContractTpl: [400,  3000],

  eMetered:     [600,  2800],
  eFixed:       [1000, 2800],
};

// ── Dati Originali Intatti ───────────────────────────────
const attrs = [
  ['eUser', 'email', true, 'top', 0], ['eUser', 'name', false, 'top', 1], ['eUser', 'role', false, 'top', 2], ['eUser', 'is_active', false, 'top', 3],
  ['eSettings', 'language', false, 'bottom', 0], ['eSettings', 'currency', false, 'bottom', 1], ['eSettings', 'theme', false, 'bottom', 2],
  ['eProperty', 'name', true, 'top', 0], ['eProperty', 'address', false, 'top', 1], ['eProperty', 'type', false, 'top', 2], ['eProperty', 'is_current', false, 'top', 3],
  ['eHMember', 'name', true, 'right', 0], ['eHMember', 'role', false, 'right', 1], ['eHMember', 'is_virtual', false, 'right', 2],
  ['eHSettings', 'split_mode', false, 'right', 0], ['eHSettings', 'default_split_type', false, 'right', 1],
  ['eCategory', 'name', true, 'left', 0], ['eCategory', 'icon', false, 'left', 1], ['eCategory', 'color', false, 'left', 2],
  ['eSubcategory', 'name', true, 'left', 0],
  ['eExpense', 'amount', false, 'top', 0], ['eExpense', 'date', false, 'top', 1], ['eExpense', 'description', false, 'top', 2], ['eExpense', 'is_split', false, 'top', 3],
  ['eSplit', 'amount', false, 'top', 0], ['eSplit', 'is_settled', false, 'top', 1],
  ['eSettlement', 'amount', false, 'right', 0], ['eSettlement', 'date', false, 'right', 1], ['eSettlement', 'payment_method', false, 'right', 2],
  ['eProject', 'name', true, 'bottom', 0], ['eProject', 'budget', false, 'bottom', 1], ['eProject', 'status', false, 'bottom', 2],
  ['eUtility', 'type', false, 'left', 0], ['eUtility', 'provider', false, 'left', 1], ['eUtility', 'service_code', true, 'left', 2], ['eUtility', 'is_active', false, 'left', 3],
  ['eReading', 'reading_date', true, 'top', 0], ['eReading', 'value', false, 'top', 1], ['eReading', 'source', false, 'top', 2],
  ['eBill', 'bill_number', true, 'bottom', 0], ['eBill', 'amount_total', false, 'bottom', 1], ['eBill', 'due_date', false, 'bottom', 2], ['eBill', 'is_paid', false, 'bottom', 3],
  ['eRate', 'effective_date', true, 'top', 0], ['eRate', 'rate_unit', false, 'top', 1],
  ['ePriceChg', 'effective_date', true, 'right', 0], ['ePriceChg', 'old_amount', false, 'right', 1], ['ePriceChg', 'new_amount', false, 'right', 2],
  ['eComm', 'type', false, 'right', 0], ['eComm', 'title', true, 'right', 1], ['eComm', 'is_important', false, 'right', 2],
  ['eBillTpl', 'name', true, 'bottom', 0], ['eBillTpl', 'provider', false, 'bottom', 1], ['eBillTpl', 'extraction_rules', false, 'bottom', 2],
  ['eContractTpl', 'name', true, 'bottom', 0], ['eContractTpl', 'provider', false, 'bottom', 1]
];

const rels = [
  ['rHa', 'ha', 'eUser', '(1,1)', 'eSettings', '(1,1)'],
  ['rPossiede', 'possiede', 'eUser', '(0,N)', 'eProperty', '(1,1)'],
  ['rInclude', 'include', 'eProperty', '(0,N)', 'eHMember', '(1,1)'],
  ['rE', 'è', 'eUser', '(0,N)', 'eHMember', '(0,1)'],
  ['rConfigura', 'configura', 'eProperty', '(0,1)', 'eHSettings', '(1,1)'],
  ['rHaCat', 'ha', 'eCategory', '(0,N)', 'eSubcategory', '(1,1)'],
  ['rRegistra', 'registra', 'eUser', '(0,N)', 'eExpense', '(1,1)'],
  ['rClassifica', 'classifica', 'eCategory', '(0,N)', 'eExpense', '(1,1)'],
  ['rPagata', 'pagata da', 'eHMember', '(0,N)', 'eExpense', '(1,1)'],
  ['rSuddivisa', 'suddivisa in', 'eExpense', '(0,N)', 'eSplit', '(1,1)'],
  ['rQuotaDi', 'quota di', 'eHMember', '(0,N)', 'eSplit', '(1,1)'],
  ['rSalda', 'salda', 'eSettlement', '(0,N)', 'eSplit', '(0,1)'],
  ['rDa', 'da', 'eHMember', '(0,N)', 'eSettlement', '(1,1)'],
  ['rA', 'a', 'eHMember', '(0,N)', 'eSettlement', '(1,1)'],
  ['rCrea', 'crea', 'eUser', '(0,N)', 'eProject', '(1,1)'],
  ['rInProg', 'in progetto', 'eProject', '(0,N)', 'eExpense', '(0,1)'],
  ['rCondiviso', 'condiviso', 'eProject', '(0,N)', 'eUser', '(0,N)'],
  ['rGestisce', 'gestisce', 'eUser', '(0,N)', 'eUtility', '(1,1)'],
  ['rServita', 'servita da', 'eProperty', '(0,N)', 'eUtility', '(1,1)'],
  ['rRileva', 'rileva', 'eUtility', '(0,N)', 'eReading', '(1,1)'],
  ['rFattura', 'fattura', 'eUtility', '(0,N)', 'eBill', '(1,1)'],
  ['rTariffa', 'tariffa', 'eUtility', '(0,N)', 'eRate', '(1,1)'],
  ['rVariazione', 'variazione', 'eUtility', '(0,N)', 'ePriceChg', '(1,1)'],
  ['rComunica', 'comunica', 'eUtility', '(0,N)', 'eComm', '(1,1)'],
  ['rUsaTpl', 'usa', 'eUtility', '(0,1)', 'eBillTpl', '(0,N)'],
  ['rAssociata', 'associata a', 'eBill', '(0,1)', 'eReading', '(0,1)'],
  ['rDaBollPc', 'da bolletta', 'ePriceChg', '(0,1)', 'eBill', '(0,N)'],
  ['rDaBollCm', 'da bolletta', 'eComm', '(0,1)', 'eBill', '(0,N)'],
  ['rCreaTpl', 'crea', 'eUser', '(0,N)', 'eBillTpl', '(1,1)'],
  ['rCreaCTpl', 'crea', 'eUser', '(0,N)', 'eContractTpl', '(1,1)']
];

// ── Helper Generazione Celle ─────────────────────────────
function vertex(cellId, value, style, x, y, w, h) {
  return `<mxCell id="${cellId}" value="${esc(value)}" style="${style}" vertex="1" parent="1"><mxGeometry x="${x}" y="${y}" width="${w}" height="${h}" as="geometry"/></mxCell>`;
}

function edgeCell(cellId, src, tgt, style) {
  return `<mxCell id="${cellId}" value="" style="${style}" edge="1" source="${src}" target="${tgt}" parent="1"><mxGeometry relative="1" as="geometry"/></mxCell>`;
}

// Etichette spostate vicine alla loro origine e con sfondo coprente
function labelCell(parentEdgeId, text, posStr, offsetX, offsetY) {
  return `<mxCell id="${id()}" value="${esc(text)}" style="${S.cardLabel}" vertex="1" connectable="0" parent="${parentEdgeId}"><mxGeometry x="${posStr}" y="0" relative="1" as="geometry"><mxPoint x="${offsetX}" y="${offsetY}" as="offset"/></mxGeometry></mxCell>`;
}

// ── Assemblaggio Struttura ───────────────────────────────
const cells = [];

// 1. Etichette di sezione
cells.push(vertex('sec1', 'UTENTI E PROPRIETÀ', S.sectionLabel, 100, 40, 300, 30));
cells.push(vertex('sec2', 'SPESE E BILANCIO', S.sectionLabel, 100, 1000, 300, 30));
cells.push(vertex('sec3', 'SERVIZI (UTENZE)', S.sectionLabel, 100, 2000, 300, 30));

// 2. Entità
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

for (const [eid, pos] of Object.entries(layout)) {
  const isGen = eid === 'eMetered' || eid === 'eFixed';
  cells.push(vertex(eid, entDef[eid], isGen ? S.genChild : S.entity, pos[0], pos[1], EW, EH));
}

// 3. Attributi (Raggio stretto per non invadere lo spazio)
const groupedAttrs = attrs.reduce((acc, curr) => {
  const key = `${curr[0]}-${curr[3]}`;
  if (!acc[key]) acc[key] = [];
  acc[key].push(curr);
  return acc;
}, {});

for (const gk in groupedAttrs) {
  groupedAttrs[gk].forEach((attr, idx) => {
    const [entId, name, isKey, side] = attr;
    const [ex, ey] = layout[entId];
    const centerX = ex + EW / 2;
    const centerY = ey + EH / 2;
    
    const spread = Math.PI * 0.7;
    const baseAngles = { top: -Math.PI / 2, bottom: Math.PI / 2, left: Math.PI, right: 0 };
    const startAngle = baseAngles[side] - spread / 2;
    const step = groupedAttrs[gk].length > 1 ? spread / (groupedAttrs[gk].length - 1) : 0;
    const angle = startAngle + (idx * step);
    
    // Raggio ridotto per mantenere compattezza
    const radius = 85 + (Math.floor(idx / 4) * 35); 
    const ax = centerX + radius * Math.cos(angle) - AW / 2;
    const ay = centerY + radius * Math.sin(angle) - AH / 2;

    const aId = id();
    cells.push(vertex(aId, name, isKey ? S.attrKey : S.attr, ax, ay, AW, AH));
    cells.push(edgeCell(id(), entId, aId, isKey ? S.lineIdent : S.line));
  });
}

// 4. Relazioni (Jitter offset per spargere i diamanti)
rels.forEach((r, i) => {
  const [rId, name, e1, c1, e2, c2] = r;
  const p1 = layout[e1]; const p2 = layout[e2];
  
  // Offset deterministico basato sulla lunghezza del nome per evitare collisioni perfette
  const offset = (name.length * 4) - 15;
  const mx = (p1[0] + p2[0]) / 2 + (EW - DW) / 2 + offset;
  const my = (p1[1] + p2[1]) / 2 + (EH - DH) / 2 + (i % 2 === 0 ? 20 : -20);

  cells.push(vertex(rId, name, S.rel, mx, my, DW, DH));

  const l1 = id();
  cells.push(edgeCell(l1, e1, rId, S.relLine));
  cells.push(labelCell(l1, c1, "-0.5", 15, -15));

  const l2 = id();
  cells.push(edgeCell(l2, rId, e2, S.relLine));
  cells.push(labelCell(l2, c2, "0.5", 15, -15));
});

// 5. Generalizzazioni
cells.push(edgeCell(id(), 'eMetered', 'eUtility', S.genArrow));
cells.push(edgeCell(id(), 'eFixed', 'eUtility', S.genArrow));

// ── Esportazione Finale ──────────────────────────────────
const xml = `<?xml version="1.0" encoding="UTF-8"?>
<mxfile host="app.diagrams.net" type="device" compressed="false">
  <diagram id="homelog-er" name="HomeLog ER (Pristine)">
    <mxGraphModel dx="4000" dy="4000" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="0" pageScale="1" pageWidth="5000" pageHeight="4000" math="0" shadow="0">
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>
${cells.join('\n')}
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
`;

writeFileSync('er-diagram.drawio', xml, 'utf8');
console.log('✅ File er-diagram.drawio rigenerato. Risolti gli attraversamenti!');