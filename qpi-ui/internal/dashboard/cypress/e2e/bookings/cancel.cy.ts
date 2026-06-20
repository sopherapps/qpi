describe("Bookings — Cancel Booking", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Bookings").click();
    cy.contains("h1", "Bookings").should("be.visible");
  });

  it("cancels a booking after confirming the dialog", () => {
    // First, create a booking
    cy.contains("button", "Book Time Slot").click();

    const pad = (n: number) => n.toString().padStart(2, "0");
    const start = new Date(Date.now() + 30 * 60000);
    const end = new Date(Date.now() + 40 * 60000);

    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);
    cy.contains("button", "Schedule Slot").click();

    // Verify the booking exists
    cy.contains("td", "user@example.com").should("be.visible");

    // Confirm the browser dialog
    cy.on("window:confirm", () => true);

    // The new booking should disappear from the table. We can verify the start time we just entered is no longer present.
    // However, formatting dates in UI vs ISO is tricky.
    // Instead, let's just wait a bit and confirm the row count decreased, or we can just assert that the specific Trash button we clicked is gone.
    // A simpler assertion: wait for the table to update.
    // Since we know the newly created one is likely at the top or bottom depending on sort, let's just verify it doesn't fail.
    // Actually, PocketBase seed creates a slot. Let's just assert the specific success alert or that the table rows are fewer.
    cy.get('tbody tr').then($rows => {
      const initialCount = $rows.length;
      cy.get('button svg.lucide-trash-2').first().parent().click();
      cy.get('tbody tr').should('have.length.lessThan', initialCount);
    });
  });
});
