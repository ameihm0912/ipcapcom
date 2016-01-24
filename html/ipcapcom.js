function pushResponse(data) {
	$('#iccform').empty();
	sendPing();
}

function purgeResponse(data) {
	$('#iccform').empty();
	sendPing();
}

function makeForm() {
	var tHTML = `
		<table>
		<tr><td>Authorization token</td><td><input name="authtoken" type="password"></input></tr></td>
		<tr><td>Duration</td><td><input name="duration" type="text"></input></tr></td>
		<tr><td><input id="submit" type="submit"></input></td><td></td></tr>
		</table>
		`;
	nf = $('<form id="iccformel"></form>').html(tHTML);
	nf.attr('action', '#');
	$('#iccform').append(nf);
	$('input#submit').click(function() {
		$.ajax({
			url: '/apply',
			type: 'post',
			dataType: 'json',
			data: $('form#iccformel').serialize(),
			success: pushResponse
		});
		return false;
	});
}

function pingParser(data) {
	if (data.no_state) {
		makeForm();
	} else {
		tHTML = '<p>Exemption is applied, expires ' + data.expires + '</p>';
		$('#iccform').html(tHTML);
		lnk = '<a id="purge" href="#">Purge exemption</a>'
		$('#iccform').append($('<p>').html(lnk));
		$('#purge').click(function(d) {
			d.preventDefault();
			$.ajax({
				url: '/purge',
				type: 'get',
				dataType: 'json',
				success: purgeResponse
			});
		});
	}
}

function sendPing() {
	$.ajax({url: "/ping", success: pingParser});
}

document.addEventListener("DOMContentLoaded", function() {
	sendPing();
});
