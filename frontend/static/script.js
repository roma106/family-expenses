const API = "/expenses";
const tbody = document.querySelector("#expenses-body");
const statusText = document.querySelector("#status");
const dialog = document.querySelector("#expense-dialog");
const form = document.querySelector("#expense-form");
const filters = document.querySelector("#filters");

let expenses = [];
let editingId = null;

function formatMoney(amount) {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency", currency: "RUB"
  }).format(Number(amount));
}

function formatDate(date) {
  // YYYY-MM-DD
  const parts = String(date).slice(0, 10).split("-");
  return parts.length === 3 ? `${parts[2]}.${parts[1]}.${parts[0]}` : date;
}

async function loadExpenses() {
  statusText.textContent = "Загрузка...";

  const params = new URLSearchParams(new FormData(filters));
  for (const [key, value] of params) {
    if (!value) params.delete(key);
  }
  const url = params.size ? `${API}?${params}` : API;

  try {
    const response = await fetch(url);
    if (!response.ok) throw new Error(`Ошибка ${response.status}`);

    expenses = await response.json();
    renderExpenses();
    statusText.textContent = "";
  } catch (error) {
    statusText.textContent = `Не удалось загрузить расходы: ${error.message}`;
  }
}

function renderExpenses() {
  tbody.replaceChildren();
  let total = 0;

  for (const expense of expenses) {
    total += Number(expense.amount);
    const tr = document.createElement("tr");

    const values = [
      formatDate(expense.spent_at),
      String(expense.user_id),
      expense.category,
      expense.comment || "—",
      formatMoney(expense.amount)
    ];

    for (let i = 0; i < values.length; i++) {
      const td = document.createElement("td");
      td.textContent = values[i];
      if (i === 2) td.className = "category-cell";
      if (i === 4) td.className = "amount";
      tr.append(td);
    }

    const actions = document.createElement("td");
    actions.className = "actions";

    const edit = document.createElement("button");
    edit.textContent = "Изменить";
    edit.className = "link-button";
    edit.onclick = () => openEdit(expense);

    const remove = document.createElement("button");
    remove.textContent = "Удалить";
    remove.className = "link-button danger";
    remove.onclick = () => deleteExpense(expense.id);

    actions.append(edit, remove);
    tr.append(actions);
    tbody.append(tr);
  }

  if (expenses.length === 0) {
    const row = document.createElement("tr");
    const cell = document.createElement("td");
    cell.colSpan = 6;
    cell.className = "empty";
    cell.textContent = "Трат пока нет";
    row.append(cell);
    tbody.append(row);
  }

  document.querySelector("#total-amount").textContent = formatMoney(total);
  document.querySelector("#expense-count").textContent = expenses.length;
  // TODO: загрузить список участников для фильтра с отдельного endpoint.
}

function openNew() {
  editingId = null;
  form.reset();
  form.elements.spent_at.value = new Date().toLocaleDateString("en-CA");
  document.querySelector("#dialog-title").textContent = "Новая трата";
  document.querySelector("#form-error").textContent = "";
  dialog.showModal();
}

function openEdit(expense) {
  editingId = expense.id;
  form.elements.user_id.value = expense.user_id;
  form.elements.amount.value = expense.amount;
  form.elements.category.value = expense.category;
  form.elements.spent_at.value = String(expense.spent_at).slice(0, 10);
  form.elements.comment.value = expense.comment || "";
  document.querySelector("#dialog-title").textContent = "Редактировать трату";
  document.querySelector("#form-error").textContent = "";
  dialog.showModal();
}

async function saveExpense(event) {
  event.preventDefault();

  const data = Object.fromEntries(new FormData(form));
  data.user_id = Number(data.user_id);
  data.amount = Number(data.amount);

  const url = editingId === null ? API : `${API}/${editingId}`;
  const method = editingId === null ? "POST" : "PUT";

  try {
    const response = await fetch(url, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data)
    });
    if (!response.ok) throw new Error(`Ошибка ${response.status}`);
    dialog.close();
    await loadExpenses();
  } catch (error) {
    document.querySelector("#form-error").textContent = `Не удалось сохранить: ${error.message}`;
  }
}

async function deleteExpense(id) {
  if (!confirm("Удалить эту трату?")) return;

  try {
    const response = await fetch(`${API}/${id}`, { method: "DELETE" });
    if (!response.ok) throw new Error(`Ошибка ${response.status}`);
    await loadExpenses();
  } catch (error) {
    statusText.textContent = `Не удалось удалить: ${error.message}`;
  }
}

document.querySelector("#add-button").onclick = openNew;
document.querySelector("#refresh-button").onclick = loadExpenses;
document.querySelector("#close-dialog").onclick = () => dialog.close();
document.querySelector("#cancel-button").onclick = () => dialog.close();
filters.onsubmit = event => { event.preventDefault(); loadExpenses(); };
form.onsubmit = saveExpense;

loadExpenses();
